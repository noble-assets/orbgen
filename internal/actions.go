// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package internal

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/noble-assets/orbiter/types/controller/action"
	"github.com/noble-assets/orbiter/types/core"
)

func (m Model) writeActionSelection(s *strings.Builder) {
	// Header
	s.WriteString(bold.Render("Orbiter Payload Generator"))
	s.WriteString("\n\n")

	// Explanation
	if len(m.actions) == 0 {
		s.WriteString("Welcome! This tool helps you build payloads for cross-chain operations.\n")
		s.WriteString(
			"To start, select if you want to add a so-called " +
				bold.Render("action") +
				" to the payload.\n\n",
		)
		s.WriteString(
			"Actions are optional operations that run before forwarding (e.g. fee payments).\n",
		)
		s.WriteString("The selected actions will be run sequentially, so bear that in mind.\n\n")
	} else {
		s.WriteString("Add another action or continue to forwarding selection.\n")
		s.WriteString("Current actions: ")
		for i, act := range m.actions {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(act.Id.String())
		}
		s.WriteString("\n\n")
	}

	// List
	s.WriteString(m.list.View())
}

func (m Model) writeFeeActionSelection(s *strings.Builder) {
	s.WriteString(bold.Render("Configure Fee Action"))
	s.WriteString("\n\n")
	s.WriteString("Fee actions allow you to collect a percentage of the transaction amount.\n")
	s.WriteString("The recipient will receive the specified percentage as a fee.\n\n")

	for _, input := range m.actionInputs {
		s.WriteString(input.View() + "\n")
	}

	s.WriteString("\nUse Tab/Shift+Tab to navigate fields, Enter to add action, Ctrl+C to quit")
}

func (m Model) initFeeActionInput() Model {
	inputs := make([]textinput.Model, 2)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Fee recipient address"
	inputs[0].CharLimit = 100
	inputs[0].Width = 50

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Basis points (e.g. 100 for 1%)"
	inputs[1].CharLimit = 5
	inputs[1].Width = 30

	m.actionInputs = inputs
	m.state = feeActionInput
	focusIndex = 0

	// Focus the first input
	m.actionInputs[0].Focus()

	return m
}

func (m Model) processFeeAction() (tea.Model, tea.Cmd) {
	recipientAddr := strings.TrimSpace(m.actionInputs[0].Value())
	basisPointsStr := strings.TrimSpace(m.actionInputs[1].Value())

	if recipientAddr == "" {
		m.err = errors.New("recipient address is required")

		return m, nil
	}
	if basisPointsStr == "" {
		m.err = errors.New("basis points is required")

		return m, nil
	}

	basisPoints, err := strconv.ParseUint(basisPointsStr, 10, 32)
	if err != nil {
		m.err = fmt.Errorf("invalid basis points: %w", err)

		return m, nil
	}

	feeAttr := action.FeeAttributes{
		FeesInfo: []*action.FeeInfo{
			{
				Recipient:   recipientAddr,
				BasisPoints: uint32(basisPoints),
			},
		},
	}

	if err = feeAttr.Validate(); err != nil {
		m.err = fmt.Errorf("invalid fee attributes: %w", err)

		return m, nil
	}

	feeAction := core.Action{
		Id: core.ACTION_FEE,
	}

	err = feeAction.SetAttributes(&feeAttr)
	if err != nil {
		m.err = fmt.Errorf("failed to set action attributes: %w", err)

		return m, nil
	}

	if err = feeAction.Validate(); err != nil {
		m.err = fmt.Errorf("invalid fee action: %w", err)

		return m, nil
	}

	m.actions = append(m.actions, &feeAction)

	return m.initActionSelection(), nil
}

func (m Model) initActionSelection() Model {
	actionItems := []list.Item{
		item{title: core.ACTION_FEE.String(), desc: "Add fee payment action"},
		item{title: core.ACTION_SWAP.String(), desc: "Add token swap action"},
		item{title: "No more actions", desc: "Proceed to forwarding selection"},
	}

	l := list.New(actionItems, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select an action to add:"

	// Apply stored window dimensions if we have them
	if m.windowWidth > 0 && m.windowHeight > 0 {
		l.SetWidth(m.windowWidth)
		l.SetHeight(m.windowHeight - 3)
	}

	m.list = l
	m.state = actionSelection

	return m
}

// initFeeActionForm creates the form input for the fee action
// inputs.
func (m *Model) initFeeActionForm() {
	m.feeActionForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("fee_action_recipient").
				Title("Recipient").
				Validate(func(input string) error {
					_, err := sdk.AccAddressFromBech32(input)
					return err
				}),
			// TODO: Add validation here as well!
			huh.NewInput().
				Key("fee_action_basis_points").
				Title("Basis Points").
				Validate(
					func(s string) error {
						bps, err := strconv.Atoi(s)
						if err != nil {
							return err
						}

						return validateBPS(bps)
					},
				),
		).
			Title("Configure Fee Action"),
	)
}

const BPSNormalizer = 10_000

// ValidateBPS validates the value used for the fee basis points.
//
// TODO:  this should be refactored on the Orbiter repo to be imported here.
func validateBPS(bps int) error {
	if bps == 0 {
		return errors.New("fee basis point cannot be zero")
	}

	if bps > BPSNormalizer {
		return fmt.Errorf("fee basis point cannot be higher than %d", BPSNormalizer)
	}

	return nil
}
