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

import "strings"

const queryValueMuted = "muted"

// HasInOperator checks if the AST contains an "in:" operator
func HasInOperator(node Node) bool {
	if node == nil {
		return false
	}

	switch n := node.(type) {
	case *Term:
		return strings.EqualFold(n.Field, "in")
	case *BinaryExpr:
		return HasInOperator(n.Left) || HasInOperator(n.Right)
	case *NotExpr:
		return HasInOperator(n.Expr)
	case *ParenExpr:
		return HasInOperator(n.Expr)
	default:
		return false
	}
}

// HasInAnywhere checks if the AST contains an "in:anywhere" operator
func HasInAnywhere(node Node) bool {
	if node == nil {
		return false
	}

	switch n := node.(type) {
	case *Term:
		if strings.EqualFold(n.Field, "in") {
			for _, value := range n.Values {
				if strings.ToLower(strings.TrimSpace(value)) == "anywhere" {
					return true
				}
			}
		}
		return false
	case *BinaryExpr:
		return HasInAnywhere(n.Left) || HasInAnywhere(n.Right)
	case *NotExpr:
		return HasInAnywhere(n.Expr)
	case *ParenExpr:
		return HasInAnywhere(n.Expr)
	default:
		return false
	}
}

// HasExplicitMuted checks if the AST explicitly asks for muted notifications
// (via is:muted or muted:true)
func HasExplicitMuted(node Node) bool {
	if node == nil {
		return false
	}

	switch n := node.(type) {
	case *Term:
		field := strings.ToLower(n.Field)
		switch field {
		case "is":
			for _, value := range n.Values {
				if strings.ToLower(strings.TrimSpace(value)) == queryValueMuted {
					return true
				}
			}
		case queryValueMuted:
			for _, value := range n.Values {
				val := strings.ToLower(strings.TrimSpace(value))
				if val == "true" || val == "yes" || val == "1" {
					return true
				}
			}
		}
		return false
	case *BinaryExpr:
		return HasExplicitMuted(n.Left) || HasExplicitMuted(n.Right)
	case *NotExpr:
		// Don't check NOT expressions - if they negate muted, we still want to exclude muted
		return false
	case *ParenExpr:
		return HasExplicitMuted(n.Expr)
	default:
		return false
	}
}
