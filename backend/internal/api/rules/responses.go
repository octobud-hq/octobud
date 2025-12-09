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

package rules

import (
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// RuleActions is the response type for rule actions
type RuleActions = models.RuleActions

// RuleResponse is the response type for a rule
type RuleResponse = models.Rule

// listRulesResponse is the response type for a list of rules
type listRulesResponse struct {
	Rules []RuleResponse `json:"rules"`
}

type ruleEnvelope struct {
	Rule RuleResponse `json:"rule"`
}
