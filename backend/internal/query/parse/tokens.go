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

import "fmt"

// TokenType represents the type of token
type TokenType int

// TokenType constants
const (
	TokenEOF      TokenType = iota
	TokenLParen             // (
	TokenRParen             // )
	TokenAnd                // AND
	TokenOr                 // OR
	TokenNot                // NOT or -
	TokenColon              // :
	TokenComma              // ,
	TokenField              // field name
	TokenValue              // field value
	TokenFreeText           // free text word
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int // Position in original string
}

// String returns a string representation of the token type
func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenLParen:
		return "LPAREN"
	case TokenRParen:
		return "RPAREN"
	case TokenAnd:
		return "AND"
	case TokenOr:
		return "OR"
	case TokenNot:
		return "NOT"
	case TokenColon:
		return "COLON"
	case TokenComma:
		return "COMMA"
	case TokenField:
		return "FIELD"
	case TokenValue:
		return "VALUE"
	case TokenFreeText:
		return "FREETEXT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

// String returns a string representation of the token
func (t Token) String() string {
	if t.Value == "" {
		return fmt.Sprintf("%s@%d", t.Type, t.Pos)
	}
	return fmt.Sprintf("%s(%q)@%d", t.Type, t.Value, t.Pos)
}
