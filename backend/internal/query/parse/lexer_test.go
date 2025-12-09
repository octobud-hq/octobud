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

func TestLexer_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []TokenType{TokenEOF},
		},
		{
			name:     "simple field:value",
			input:    "repo:cli",
			expected: []TokenType{TokenFreeText, TokenColon, TokenFreeText, TokenEOF},
		},
		{
			name:  "parentheses",
			input: "(repo:cli)",
			expected: []TokenType{
				TokenLParen,
				TokenFreeText,
				TokenColon,
				TokenFreeText,
				TokenRParen,
				TokenEOF,
			},
		},
		{
			name:  "comma separated values",
			input: "repo:cli,other",
			expected: []TokenType{
				TokenFreeText,
				TokenColon,
				TokenFreeText,
				TokenComma,
				TokenFreeText,
				TokenEOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, tok := range tokens {
				if tok.Type != tt.expected[i] {
					t.Errorf("token %d: expected type %s, got %s", i, tt.expected[i], tok.Type)
				}
			}
		})
	}
}

func TestLexer_Operators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:  "AND operator",
			input: "repo:cli AND is:unread",

			expected: []TokenType{
				TokenFreeText,
				TokenColon,
				TokenFreeText,
				TokenAnd,
				TokenFreeText,
				TokenColon,
				TokenFreeText,
				TokenEOF,
			},
		},
		{
			name:  "OR operator",
			input: "repo:cli OR repo:other",

			expected: []TokenType{
				TokenFreeText,
				TokenColon,
				TokenFreeText,
				TokenOr,
				TokenFreeText,
				TokenColon,
				TokenFreeText,
				TokenEOF,
			},
		},
		{
			name:     "NOT operator",
			input:    "NOT repo:cli",
			expected: []TokenType{TokenNot, TokenFreeText, TokenColon, TokenFreeText, TokenEOF},
		},
		{
			name:     "dash NOT operator",
			input:    "-repo:cli",
			expected: []TokenType{TokenNot, TokenFreeText, TokenColon, TokenFreeText, TokenEOF},
		},
		{
			name:     "lowercase operators are free text",
			input:    "and or not",
			expected: []TokenType{TokenFreeText, TokenFreeText, TokenFreeText, TokenEOF},
		},
		{
			name:     "uppercase operators",
			input:    "AND OR NOT",
			expected: []TokenType{TokenAnd, TokenOr, TokenNot, TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, tok := range tokens {
				if tok.Type != tt.expected[i] {
					t.Errorf(
						"token %d: expected type %s, got %s (value: %q)",
						i,
						tt.expected[i],
						tok.Type,
						tok.Value,
					)
				}
			}
		})
	}
}

func TestLexer_QuotedStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple quoted string",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "quoted string with special chars",
			input:    `"fix: urgent (P0)"`,
			expected: "fix: urgent (P0)",
		},
		{
			name:     "quoted string with escaped quote",
			input:    `"say \"hello\""`,
			expected: `say "hello"`,
		},
		{
			name:     "quoted string with escaped backslash",
			input:    `"path\\to\\file"`,
			expected: `path\to\file`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) < 1 {
				t.Fatal("expected at least one token")
			}

			if tokens[0].Type != TokenValue {
				t.Errorf("expected TokenValue, got %s", tokens[0].Type)
			}

			if tokens[0].Value != tt.expected {
				t.Errorf("expected value %q, got %q", tt.expected, tokens[0].Value)
			}
		})
	}
}

func TestLexer_SpecialChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "hyphenated word",
			input: "some-word",
			valid: true,
		},
		{
			name:  "slash in value",
			input: "repo:github/cli",
			valid: true,
		},
		{
			name:  "dots in value",
			input: "file:config.yaml",
			valid: true,
		},
		{
			name:  "at sign in value",
			input: "email:user@example.com",
			valid: true,
		},
		{
			name:  "brackets in value for bot names",
			input: "author:dependabot[bot]",
			valid: true,
		},
		{
			name:  "brackets only in value",
			input: "author:[bot]",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if tt.valid {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if len(tokens) < 2 {
					t.Error("expected tokens to be generated")
				}
			} else if err == nil {
				t.Error("expected error, got none")
			}
		})
	}
}

func TestLexer_ErrorCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unterminated quoted string",
			input: `"hello`,
		},
		{
			name:  "unterminated quoted string with escape",
			input: `"hello\"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			_, err := lexer.Tokenize()
			if err == nil {
				t.Error("expected error, got none")
			}
		})
	}
}

func TestLexer_ComplexQueries(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "complex query with grouping",
			input: "(repo:cli OR repo:other) AND is:unread",
		},
		{
			name:  "negation with comma values",
			input: "-author:bot,dependabot",
		},
		{
			name:  "mixed operators and quoted strings",
			input: `repo:cli AND subject:"urgent fix" OR in:snoozed`,
		},
		{
			name:  "deeply nested",
			input: "((repo:cli AND is:unread) OR (in:snoozed AND repo:other)) AND NOT author:bot",
		},
		{
			name:  "exclude bot authors with brackets",
			input: "-author:[bot]",
		},
		{
			name:  "multiple bot exclusions with brackets",
			input: "-author:dependabot[bot],renovate[bot]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Just verify we can tokenize complex queries without errors
			if len(tokens) == 0 {
				t.Error("expected non-empty token list")
			}

			if tokens[len(tokens)-1].Type != TokenEOF {
				t.Error("expected last token to be EOF")
			}
		})
	}
}
