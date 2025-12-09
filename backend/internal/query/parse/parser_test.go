// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package parse

import (
	"testing"
)

func TestParser_BasicTerms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // String representation of AST
	}{
		{
			name:     "simple term",
			input:    "repo:cli",
			expected: "repo:cli",
		},
		{
			name:     "comma separated values",
			input:    "repo:cli,other",
			expected: "repo:cli,other",
		},
		{
			name:     "quoted value",
			input:    `subject:"urgent fix"`,
			expected: `subject:urgent fix`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			if ast == nil {
				t.Fatal("expected non-nil AST")
			}

			if ast.String() != tt.expected {
				t.Errorf("expected AST %q, got %q", tt.expected, ast.String())
			}
		})
	}
}

func TestParser_BinaryOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "explicit AND",
			input:    "repo:cli AND is:unread",
			expected: "(repo:cli AND is:unread)",
		},
		{
			name:     "explicit OR",
			input:    "repo:cli OR repo:other",
			expected: "(repo:cli OR repo:other)",
		},
		{
			name:     "implicit AND (whitespace)",
			input:    "repo:cli is:unread",
			expected: "(repo:cli AND is:unread)",
		},
		{
			name:     "mixed operators",
			input:    "repo:cli OR repo:other AND is:unread",
			expected: "(repo:cli OR (repo:other AND is:unread))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			if ast.String() != tt.expected {
				t.Errorf("expected AST %q, got %q", tt.expected, ast.String())
			}
		})
	}
}

func TestParser_Precedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "OR has lower precedence than AND",
			input:    "a:1 OR b:2 AND c:3",
			expected: "(a:1 OR (b:2 AND c:3))",
		},
		{
			name:     "AND has lower precedence than NOT",
			input:    "NOT a:1 AND b:2",
			expected: "(NOT(a:1) AND b:2)",
		},
		{
			name:     "multiple ANDs left-to-right",
			input:    "a:1 AND b:2 AND c:3",
			expected: "((a:1 AND b:2) AND c:3)",
		},
		{
			name:     "multiple ORs left-to-right",
			input:    "a:1 OR b:2 OR c:3",
			expected: "((a:1 OR b:2) OR c:3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			if ast.String() != tt.expected {
				t.Errorf("expected AST %q, got %q", tt.expected, ast.String())
			}
		})
	}
}

func TestParser_Grouping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple parens",
			input:    "(repo:cli)",
			expected: "(repo:cli)",
		},
		{
			name:     "parens override precedence",
			input:    "(repo:cli OR repo:other) AND is:unread",
			expected: "(((repo:cli OR repo:other)) AND is:unread)",
		},
		{
			name:     "nested parens",
			input:    "((a:1 OR b:2) AND c:3)",
			expected: "((((a:1 OR b:2)) AND c:3))",
		},
		{
			name:     "multiple groups",
			input:    "(a:1 OR b:2) AND (c:3 OR d:4)",
			expected: "(((a:1 OR b:2)) AND ((c:3 OR d:4)))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			if ast.String() != tt.expected {
				t.Errorf("expected AST %q, got %q", tt.expected, ast.String())
			}
		})
	}
}

func TestParser_Negation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "NOT operator",
			input:    "NOT repo:cli",
			expected: "NOT(repo:cli)",
		},
		{
			name:     "dash NOT",
			input:    "-repo:cli",
			expected: "NOT(repo:cli)",
		},
		{
			name:     "NOT with grouping",
			input:    "NOT (repo:cli OR repo:other)",
			expected: "NOT(((repo:cli OR repo:other)))",
		},
		{
			name:     "double NOT",
			input:    "NOT NOT repo:cli",
			expected: "NOT(NOT(repo:cli))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			if ast.String() != tt.expected {
				t.Errorf("expected AST %q, got %q", tt.expected, ast.String())
			}
		})
	}
}

func TestParser_FreeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single word",
			input:    "urgent",
			expected: `FREE("urgent")`,
		},
		{
			name:     "free text with term",
			input:    "repo:cli urgent",
			expected: `(repo:cli AND FREE("urgent"))`,
		},
		{
			name:     "multiple free text words",
			input:    "urgent fix",
			expected: `(FREE("urgent") AND FREE("fix"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			if ast.String() != tt.expected {
				t.Errorf("expected AST %q, got %q", tt.expected, ast.String())
			}
		})
	}
}

func TestParser_ComplexQueries(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "deeply nested",
			input: "((repo:cli AND is:unread) OR (in:snoozed AND repo:other)) AND NOT author:bot",
		},
		{
			name:  "comma OR with explicit operators",
			input: "repo:cli,other AND (is:read OR is:unread)",
		},
		{
			name:  "mixed free text and terms",
			input: `(repo:cli OR repo:other) AND "urgent fix"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			if ast == nil {
				t.Fatal("expected non-nil AST")
			}

			// Just verify we can parse complex queries without errors
			_ = ast.String()
		})
	}
}

func TestParser_ErrorCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unbalanced parens (missing close)",
			input: "(repo:cli",
		},
		{
			name:  "unbalanced parens (missing open)",
			input: "repo:cli)",
		},
		{
			name:  "colon without field",
			input: ":value",
		},
		{
			name:  "colon without value",
			input: "repo:",
		},
		{
			name:  "empty parens",
			input: "()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				// Lexer error is acceptable
				return
			}

			parser := NewParser(tokens)
			_, err = parser.Parse()
			if err == nil {
				t.Error("expected parser error, got none")
			}
		})
	}
}
