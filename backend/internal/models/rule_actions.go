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

package models

// RuleActions represents actions that a rule can perform
type RuleActions struct {
	SkipInbox  bool     `json:"skipInbox"`
	MarkRead   bool     `json:"markRead,omitempty"`
	Star       bool     `json:"star,omitempty"`
	Archive    bool     `json:"archive,omitempty"`
	Mute       bool     `json:"mute,omitempty"`
	AssignTags []string `json:"assignTags,omitempty"`
	RemoveTags []string `json:"removeTags,omitempty"`
}
