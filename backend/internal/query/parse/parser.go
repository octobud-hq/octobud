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
	"errors"
	"fmt"
)

// Error definitions
var (
	ErrUnexpectedTokenAfterExpression = errors.New("unexpected token after expression")
	ErrUnexpectedClosingParen         = errors.New("unexpected closing parenthesis")
	ErrUnexpectedToken                = errors.New("unexpected token")
	ErrExpectedOpeningParen           = errors.New("expected opening parenthesis")
	ErrExpectedClosingParen           = errors.New("expected closing parenthesis")
	ErrExpectedFieldName              = errors.New("expected field name")
	ErrExpectedColon                  = errors.New("expected colon after field")
	ErrExpectedValue                  = errors.New("expected value after colon")
	ErrExpectedAtLeastOneValue        = errors.New("expected at least one value for field")
)

// Parser parses tokens into an Abstract Syntax Tree
type Parser struct {
	tokens  []Token
	pos     int
	current Token
}

// NewParser creates a new parser for the given tokens
func NewParser(tokens []Token) *Parser {
	p := &Parser{
		tokens: tokens,
		pos:    0,
	}
	if len(tokens) > 0 {
		p.current = tokens[0]
	}
	return p
}

// Parse parses the tokens and returns the root AST node
func (p *Parser) Parse() (Node, error) {
	if p.current.Type == TokenEOF {
		// Empty query - return nil or a placeholder
		return nil, nil
	}

	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check for unexpected tokens after the expression
	if p.current.Type != TokenEOF {
		return nil, errors.Join(
			ErrUnexpectedTokenAfterExpression,
			fmt.Errorf("token %s at position %d", p.current.Type, p.current.Pos),
		)
	}

	return node, nil
}

// parseExpression is the entry point for parsing
func (p *Parser) parseExpression() (Node, error) {
	return p.parseOrExpr()
}

// parseOrExpr handles OR operations (lowest precedence)
func (p *Parser) parseOrExpr() (Node, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenOr {
		p.advance() // consume OR
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Op:    "OR",
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseAndExpr handles AND operations and implicit AND (whitespace)
func (p *Parser) parseAndExpr() (Node, error) {
	left, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	// Continue collecting AND operands
	for {
		// Explicit AND
		if p.current.Type == TokenAnd {
			p.advance() // consume AND
			right, err := p.parseNotExpr()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{
				Op:    "AND",
				Left:  left,
				Right: right,
			}
			continue
		}

		// Implicit AND: if we see another primary expression (not OR, not EOF, not rparen)
		if p.isStartOfPrimary() {
			right, err := p.parseNotExpr()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{
				Op:    "AND",
				Left:  left,
				Right: right,
			}
			continue
		}

		break
	}

	return left, nil
}

// parseNotExpr handles NOT operations
func (p *Parser) parseNotExpr() (Node, error) {
	if p.current.Type == TokenNot {
		p.advance()                   // consume NOT
		expr, err := p.parseNotExpr() // Allow chaining: NOT NOT expr
		if err != nil {
			return nil, err
		}
		return &NotExpr{Expr: expr}, nil
	}

	return p.parsePrimary()
}

// parsePrimary handles parentheses, terms, and free text
func (p *Parser) parsePrimary() (Node, error) {
	// Parenthesized expression
	if p.current.Type == TokenLParen {
		return p.parseParenExpr()
	}

	// Check for unexpected closing paren
	if p.current.Type == TokenRParen {
		return nil, errors.Join(
			ErrUnexpectedClosingParen,
			fmt.Errorf("at position %d", p.current.Pos),
		)
	}

	// Try to parse as a term (field:value)
	if p.isStartOfTerm() {
		return p.parseTerm()
	}

	// Free text (including quoted strings)
	if p.current.Type == TokenFreeText || p.current.Type == TokenValue {
		text := p.current.Value
		p.advance()
		return &FreeText{Text: text}, nil
	}

	return nil, errors.Join(
		ErrUnexpectedToken,
		fmt.Errorf("token %s at position %d", p.current.Type, p.current.Pos),
	)
}

// parseParenExpr parses a parenthesized expression
func (p *Parser) parseParenExpr() (Node, error) {
	if p.current.Type != TokenLParen {
		return nil, errors.Join(
			ErrExpectedOpeningParen,
			fmt.Errorf("at position %d", p.current.Pos),
		)
	}
	p.advance() // consume '('

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current.Type != TokenRParen {
		return nil, errors.Join(
			ErrExpectedClosingParen,
			fmt.Errorf("at position %d, got %s", p.current.Pos, p.current.Type),
		)
	}
	p.advance() // consume ')'

	return &ParenExpr{Expr: expr}, nil
}

// parseTerm parses a field:value1,value2 term
func (p *Parser) parseTerm() (*Term, error) {
	if p.current.Type != TokenFreeText {
		return nil, errors.Join(ErrExpectedFieldName, fmt.Errorf("at position %d", p.current.Pos))
	}

	field := p.current.Value
	p.advance()

	if p.current.Type != TokenColon {
		return nil, errors.Join(
			ErrExpectedColon,
			fmt.Errorf("after field %q at position %d", field, p.current.Pos),
		)
	}
	p.advance() // consume ':'

	// Parse values (comma-separated)
	var values []string

	for {
		// Values can be TokenFreeText or TokenValue (quoted strings)
		if p.current.Type != TokenFreeText && p.current.Type != TokenValue {
			return nil, errors.Join(
				ErrExpectedValue,
				fmt.Errorf("at position %d, got %s", p.current.Pos, p.current.Type),
			)
		}

		values = append(values, p.current.Value)
		p.advance()

		// Check for comma (OR within field)
		if p.current.Type == TokenComma {
			p.advance() // consume ','
			continue
		}

		break
	}

	if len(values) == 0 {
		return nil, errors.Join(ErrExpectedAtLeastOneValue, fmt.Errorf("for field %q", field))
	}

	return &Term{
		Field:  field,
		Values: values,
	}, nil
}

// isStartOfPrimary checks if current token can start a primary expression
func (p *Parser) isStartOfPrimary() bool {
	return p.current.Type == TokenLParen ||
		p.current.Type == TokenFreeText ||
		p.current.Type == TokenNot
}

// isStartOfTerm checks if current position starts a term (field:value)
func (p *Parser) isStartOfTerm() bool {
	if p.current.Type != TokenFreeText {
		return false
	}

	// Look ahead for colon
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1].Type == TokenColon
	}

	return false
}

// advance moves to the next token
func (p *Parser) advance() {
	p.pos++
	if p.pos < len(p.tokens) {
		p.current = p.tokens[p.pos]
	} else {
		p.current = Token{Type: TokenEOF}
	}
}
