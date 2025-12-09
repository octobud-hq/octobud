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
)

// Error definitions
var (
	ErrValidationFailed = errors.New("validation failed")
)

// Validator validates query AST nodes
type Validator struct {
	errors []string
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		errors: []string{},
	}
}

// Validate validates an AST node and returns an error if invalid
func (v *Validator) Validate(node Node) error {
	if node == nil {
		return nil
	}

	v.errors = []string{}
	v.validateNode(node)

	if len(v.errors) > 0 {
		return errors.Join(
			ErrValidationFailed,
			fmt.Errorf("errors: %s", strings.Join(v.errors, "; ")),
		)
	}

	return nil
}

// validateNode recursively validates a node
func (v *Validator) validateNode(node Node) {
	switch n := node.(type) {
	case *BinaryExpr:
		v.validateBinaryExpr(n)
	case *NotExpr:
		v.validateNotExpr(n)
	case *Term:
		v.validateTerm(n)
	case *FreeText:
		// Free text is always valid
	case *ParenExpr:
		v.validateNode(n.Expr)
	}
}

// validateBinaryExpr validates a binary expression
func (v *Validator) validateBinaryExpr(node *BinaryExpr) {
	v.validateNode(node.Left)
	v.validateNode(node.Right)
}

// validateNotExpr validates a NOT expression
func (v *Validator) validateNotExpr(node *NotExpr) {
	v.validateNode(node.Expr)
}

// validateTerm validates a term (field:value)
func (v *Validator) validateTerm(node *Term) {
	field := strings.ToLower(strings.TrimSpace(node.Field))

	// Check if field is known
	if !isKnownField(field) {
		v.errors = append(v.errors, fmt.Sprintf("unknown field: %s", field))
		return
	}

	// Validate field-specific values
	switch field {
	case "in":
		v.validateInValues(node.Values)
	case "is":
		v.validateIsValues(node.Values)
	case "read", "archived", "muted", "snoozed", "filtered":
		v.validateBooleanValues(field, node.Values)
	}
}

// validateInValues validates values for the in: operator
func (v *Validator) validateInValues(values []string) {
	validValues := map[string]bool{
		"inbox":    true,
		"archive":  true,
		"snoozed":  true,
		"filtered": true,
		"anywhere": true,
	}

	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !validValues[value] {
			v.errors = append(
				v.errors,
				fmt.Sprintf(
					"invalid value for in: operator: %s (valid: inbox, archive, snoozed, filtered, anywhere)",
					value,
				),
			)
		}
	}
}

// validateIsValues validates values for the is: operator
func (v *Validator) validateIsValues(values []string) {
	validValues := map[string]bool{
		"unread":   true,
		"read":     true,
		"archived": true,
		"muted":    true,
		"snoozed":  true,
		"starred":  true,
		"filtered": true,
	}

	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !validValues[value] {
			v.errors = append(
				v.errors,
				fmt.Sprintf(
					"invalid value for is: operator: %s (valid: unread, read, archived, muted, snoozed, starred, filtered)",
					value,
				),
			)
		}
	}
}

// validateBooleanValues validates boolean values
func (v *Validator) validateBooleanValues(field string, values []string) {
	validValues := map[string]bool{
		"true":  true,
		"false": true,
		"yes":   true,
		"no":    true,
		"1":     true,
		"0":     true,
	}

	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !validValues[value] {
			v.errors = append(
				v.errors,
				fmt.Sprintf(
					"invalid boolean value for %s: %s (valid: true, false, yes, no, 1, 0)",
					field,
					value,
				),
			)
		}
	}
}

// isKnownField checks if a field name is supported
func isKnownField(field string) bool {
	knownFields := map[string]bool{
		"in":           true,
		"is":           true,
		"repo":         true,
		"repository":   true,
		"org":          true,
		"reason":       true,
		"type":         true,
		"subject_type": true,
		"author":       true,
		"state":        true,
		"read":         true,
		"archived":     true,
		"muted":        true,
		"snoozed":      true,
		"filtered":     true,
		"tags":         true,
	}

	return knownFields[field]
}
