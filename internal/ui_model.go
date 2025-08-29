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
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/noble-assets/orbiter/types/controller/action"
	"github.com/noble-assets/orbiter/types/core"
)

// state is a toggle for the currently selected UI state.
type state int

const (
	actionSelection state = iota
	feeActionInput

	forwardingSelection
	cctpForwardingInput
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

var focusIndex int

// Model contains all relevant information and state
// for the UI to interactively build an Orbiter payload.
type Model struct {
	state state
	list  list.Model

	actionInputs     []textinput.Model
	forwardingInputs []textinput.Model

	actions    []*core.Action
	forwarding *core.Forwarding
	err        error
	payload    string

	windowWidth  int
	windowHeight int

	feeActionForm *huh.Form
}

// InitialModel creates the default view for the payload generator,
// that is shown when starting the tool.
func InitialModel() Model {
	actionItems := []list.Item{
		item{title: core.ACTION_FEE.String(), desc: "Add fee payment action"},
		item{title: core.ACTION_SWAP.String(), desc: "Add token swap action"},
		item{title: "No more actions", desc: "Proceed to forwarding selection"},
	}

	l := list.New(actionItems, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select an action to add:" //nolint:goconst

	m := Model{
		state:   actionSelection,
		list:    l,
		actions: []*core.Action{},
	}

	m.initFeeActionForm()

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.feeActionForm.Init())
}

func (m Model) GetPayload() string {
	return m.payload
}

// Update handles the different TUI states through the different
// selection modals.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == feeActionInput {
		var cmds []tea.Cmd

		res, cmd := m.feeActionForm.Update(msg)
		if newFeeActionForm, ok := res.(*huh.Form); ok {
			m.feeActionForm = newFeeActionForm
		}
		cmds = append(cmds, cmd)

		if m.feeActionForm.State == huh.StateCompleted {
			temp := core.Action{Id: core.ACTION_FEE}
			temp.SetAttributes(&action.FeeAttributes{
				FeesInfo: []*action.FeeInfo{
					{
						Recipient:   m.feeActionForm.GetString("fee_action_recipient"),
						BasisPoints: uint32(m.feeActionForm.GetInt("fee_action_basis_points")),
					},
				},
			})

			m.state = actionSelection
			m.actions = append(m.actions, &temp)

			m.initFeeActionForm()
			cmds = append(cmds, m.feeActionForm.Init())
		} else if m.feeActionForm.State == huh.StateAborted {
			m.state = actionSelection

			m.initFeeActionForm()
			cmds = append(cmds, m.feeActionForm.Init())
		}

		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		}
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 8)

		return m, nil
	}

	var cmd tea.Cmd
	switch m.state {
	case actionSelection, forwardingSelection:
		m.list, cmd = m.list.Update(msg)
	case cctpForwardingInput:
		cmd = m.updateForwardingInputs(msg)
	default:
		panic(fmt.Errorf("unhandled state: %v", m.state))
	}

	return m, cmd
}

func (m Model) View() string {
	var s strings.Builder

	switch m.state {
	case actionSelection:
		m.writeActionSelection(&s)
	case feeActionInput:
		return m.feeActionForm.View()

	case forwardingSelection:
		m.writeForwardingSelection(&s)
	case cctpForwardingInput:
		m.writeCCTPForwardingSelection(&s)
	}

	if m.err != nil {
		s.WriteString(
			errorStyle.Render("\nError: " + m.err.Error()),
		)
	}

	return s.String()
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case actionSelection:
		selected, ok := m.list.SelectedItem().(item)
		if !ok {
			panic(fmt.Sprintf("failed to cast list item to item; got: %T", m.list.SelectedItem()))
		}

		switch selected.title {
		case core.ACTION_FEE.String():
			return m.initFeeActionInput(), nil
		case core.ACTION_SWAP.String():
			panic("not implemented yet: " + core.ACTION_SWAP.String())
		case "No more actions":
			return m.initForwardingSelection(), nil
		}
	case feeActionInput:
		return m.processFeeAction()
	case forwardingSelection:
		selected, ok := m.list.SelectedItem().(item)
		if !ok {
			panic(fmt.Sprintf("failed to cast list item to item; got: %T", m.list.SelectedItem()))
		}

		switch selected.title {
		case core.PROTOCOL_CCTP.String():
			return m.initCCTPForwardingInput(), nil
		case core.PROTOCOL_IBC.String():
			panic("not supported yet: " + core.PROTOCOL_IBC.String())
		case core.PROTOCOL_HYPERLANE.String():
			panic("not implemented yet: " + core.PROTOCOL_HYPERLANE.String())
		}
	case cctpForwardingInput:
		return m.processCCTPForwarding()
	}

	return m, nil
}
