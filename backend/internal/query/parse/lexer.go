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
	"strings"
	"unicode"
)

// Error definitions
var (
	ErrUnexpectedCharacter      = errors.New("unexpected character")
	ErrUnterminatedQuotedString = errors.New("unterminated quoted string")
)

// Lexer tokenizes a query string
type Lexer struct {
	input string
	pos   int
	ch    rune
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input: input,
		pos:   0,
	}
	l.readChar()
	return l
}

// Tokenize returns all tokens from the input
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token

	for {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}

	return tokens, nil
}

// nextToken returns the next token from the input
func (l *Lexer) nextToken() (Token, error) {
	l.skipWhitespace()

	pos := l.pos

	switch l.ch {
	case 0:
		return Token{Type: TokenEOF, Pos: pos}, nil
	case '(':
		l.readChar()
		return Token{Type: TokenLParen, Value: "(", Pos: pos}, nil
	case ')':
		l.readChar()
		return Token{Type: TokenRParen, Value: ")", Pos: pos}, nil
	case ':':
		l.readChar()
		return Token{Type: TokenColon, Value: ":", Pos: pos}, nil
	case ',':
		l.readChar()
		return Token{Type: TokenComma, Value: ",", Pos: pos}, nil
	case '"':
		// Quoted string
		str, err := l.readQuotedString()
		if err != nil {
			return Token{}, err
		}
		return Token{Type: TokenValue, Value: str, Pos: pos}, nil
	case '-':
		// Could be NOT operator or part of a word/value
		// If at start of token and followed by a letter (like -repo:), it's NOT
		// If followed by whitespace or special char, it's also NOT
		// Otherwise it's part of a hyphenated word
		nextChar := l.peekChar()
		if nextChar == 0 || unicode.IsSpace(nextChar) || nextChar == '(' ||
			unicode.IsLetter(nextChar) {
			l.readChar()
			return Token{Type: TokenNot, Value: "-", Pos: pos}, nil
		}
		// It's part of a word (like in "some-word" mid-word)
		word := l.readWord()
		return l.classifyWord(word, pos), nil
	default:
		if isWordChar(l.ch) {
			word := l.readWord()
			return l.classifyWord(word, pos), nil
		}
		return Token{}, errors.Join(
			ErrUnexpectedCharacter,
			fmt.Errorf("character %q at position %d", l.ch, pos),
		)
	}
}

// readChar reads the next character from input
func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = rune(l.input[l.pos])
	}
	l.pos++
}

// peekChar returns the next character without advancing
func (l *Lexer) peekChar() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos])
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for unicode.IsSpace(l.ch) {
		l.readChar()
	}
}

// readWord reads a word (letters, digits, hyphens, underscores, slashes, dots)
func (l *Lexer) readWord() string {
	start := l.pos - 1
	for isWordChar(l.ch) {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

// readQuotedString reads a quoted string
func (l *Lexer) readQuotedString() (string, error) {
	start := l.pos
	l.readChar() // skip opening quote

	var result strings.Builder
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			if l.ch == 0 {
				return "", errors.Join(
					ErrUnterminatedQuotedString,
					fmt.Errorf("at position %d", start),
				)
			}
			// Handle escape sequences
			switch l.ch {
			case 'n':
				result.WriteRune('\n')
			case 't':
				result.WriteRune('\t')
			case '"':
				result.WriteRune('"')
			case '\\':
				result.WriteRune('\\')
			default:
				result.WriteRune(l.ch)
			}
		} else {
			result.WriteRune(l.ch)
		}
		l.readChar()
	}

	if l.ch == 0 {
		return "", errors.Join(ErrUnterminatedQuotedString, fmt.Errorf("at position %d", start))
	}

	l.readChar() // skip closing quote
	return result.String(), nil
}

// classifyWord classifies a word as an operator or field/value/freetext
func (l *Lexer) classifyWord(word string, pos int) Token {
	switch word {
	case "AND":
		return Token{Type: TokenAnd, Value: word, Pos: pos}
	case "OR":
		return Token{Type: TokenOr, Value: word, Pos: pos}
	case "NOT":
		return Token{Type: TokenNot, Value: word, Pos: pos}
	default:
		// Will be classified as field, value, or freetext by parser context
		return Token{Type: TokenFreeText, Value: word, Pos: pos}
	}
}

// isWordChar returns true if the character is part of a word
func isWordChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '-' || ch == '_' || ch == '/' ||
		ch == '.' ||
		ch == '@' ||
		ch == '[' || ch == ']'
}
