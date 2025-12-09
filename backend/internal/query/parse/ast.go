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

// Package parse provides the AST for the query language.
package parse

import (
	"fmt"
	"strings"
)

// Node is the base interface for all AST nodes
type Node interface {
	String() string // For debugging
}

// BinaryExpr represents AND/OR operations
type BinaryExpr struct {
	Op    string // "AND" or "OR"
	Left  Node
	Right Node
}

func (b *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Op, b.Right.String())
}

// NotExpr represents negation
type NotExpr struct {
	Expr Node
}

func (n *NotExpr) String() string {
	return fmt.Sprintf("NOT(%s)", n.Expr.String())
}

// Term represents field:value1,value2
type Term struct {
	Field   string
	Values  []string
	Negated bool // For -field:value syntax (deprecated in favor of NOT)
}

func (t *Term) String() string {
	prefix := ""
	if t.Negated {
		prefix = "-"
	}
	if len(t.Values) == 1 {
		return fmt.Sprintf("%s%s:%s", prefix, t.Field, t.Values[0])
	}
	return fmt.Sprintf("%s%s:%s", prefix, t.Field, strings.Join(t.Values, ","))
}

// FreeText represents unstructured search text
type FreeText struct {
	Text string
}

func (f *FreeText) String() string {
	return fmt.Sprintf("FREE(%q)", f.Text)
}

// ParenExpr represents grouped expression
type ParenExpr struct {
	Expr Node
}

func (p *ParenExpr) String() string {
	return fmt.Sprintf("(%s)", p.Expr.String())
}
