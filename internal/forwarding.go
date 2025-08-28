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
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/noble-assets/orbiter/testutil"
	"github.com/noble-assets/orbiter/types/controller/forwarding"
	"github.com/noble-assets/orbiter/types/core"
)

func (m Model) writeForwardingSelection(s *strings.Builder) {
	// Header
	s.WriteString(bold.Render("Select Forwarding Protocol"))
	s.WriteString("\n\n")

	// Explanation
	s.WriteString("Now choose how to forward your transaction to the destination chain.\n")
	s.WriteString("Each protocol supports different chains and tokens:\n\n")

	// List
	s.WriteString(m.list.View())
}

func (m Model) writeCCTPForwardingSelection(s *strings.Builder) {
	s.WriteString(bold.Render("Configure CCTP Forwarding"))
	s.WriteString("\n\n")
	s.WriteString("CCTP enables USDC transfers across chains. Configure the destination details:\n")
	s.WriteString(
		"• Domain: Chain identifier (0=Ethereum, 1=Avalanche, 2=OP, 3=Arbitrum, 6=Base)\n",
	)
	s.WriteString("• Mint Recipient: Address that receives USDC on destination\n")
	s.WriteString("• Destination Caller: Address that can call functions on destination\n")
	s.WriteString("• Passthrough Payload: Additional data to pass through (optional)\n\n")

	for _, input := range m.forwardingInputs {
		s.WriteString(input.View() + "\n")
	}

	s.WriteString("\nUse Tab/Shift+Tab to navigate fields, Enter to create payload, Ctrl+C to quit")
}

func (m Model) initForwardingSelection() Model {
	forwardingItems := []list.Item{
		item{
			title: core.PROTOCOL_CCTP.String(),
			desc:  "Circle's Cross-Chain Transfer Protocol (USDC transfers)",
		},
		item{
			title: core.PROTOCOL_IBC.String(),
			desc:  "Inter-Blockchain Communication (Cosmos ecosystem)",
		},
		item{title: core.PROTOCOL_HYPERLANE.String(), desc: "Hyperlane interchain protocol"},
	}

	l := list.New(forwardingItems, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a protocol:"

	// Apply stored window dimensions if we have them
	if m.windowWidth > 0 && m.windowHeight > 0 {
		l.SetWidth(m.windowWidth)
		l.SetHeight(m.windowHeight - 3)
	}

	m.list = l
	m.state = forwardingSelection

	return m
}

func (m Model) initCCTPForwardingInput() Model {
	inputs := make([]textinput.Model, 4)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Destination domain (e.g. 0)"
	inputs[0].CharLimit = 10
	inputs[0].Width = 30

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Mint recipient (prefix with '0x' for Hex input; otherwise base64 is assumed; put 'r' for random)"
	inputs[1].CharLimit = 128
	inputs[1].Width = 70

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Destination caller (prefix with '0x' for Hex input; otherwise base64 is assumed; put 'r' for random)"
	inputs[2].CharLimit = 128
	inputs[2].Width = 70

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Passthrough payload (can be left empty)"
	inputs[3].CharLimit = 256
	inputs[3].Width = 70

	m.forwardingInputs = inputs
	m.state = cctpForwardingInput
	focusIndex = 0

	// Focus the first input
	m.forwardingInputs[0].Focus()

	return m
}

func (m Model) processCCTPForwarding() (tea.Model, tea.Cmd) {
	domainStr := strings.TrimSpace(m.forwardingInputs[0].Value())
	mintRecipientStr := strings.TrimSpace(m.forwardingInputs[1].Value())
	destCallerStr := strings.TrimSpace(m.forwardingInputs[2].Value())
	passthroughStr := strings.TrimSpace(m.forwardingInputs[3].Value())

	if domainStr == "" {
		return m, nil
	}

	domain, err := strconv.ParseUint(domainStr, 10, 32)
	if err != nil {
		m.err = fmt.Errorf("invalid destination domain: %w", err)

		return m, nil
	}

	if mintRecipientStr == "" {
		m.err = errors.New("mint recipient cannot be empty")

		return m, nil
	}

	var mintRecipient []byte
	if mintRecipientStr == "r" {
		mintRecipient = testutil.RandomBytes(32)
	} else {
		mintRecipient, err = decodeHexOrBase64To32Bytes(mintRecipientStr)
		if err != nil {
			m.err = fmt.Errorf("invalid mint recipient: %w", err)

			return m, nil
		}
	}

	var destCaller []byte
	if destCallerStr == "r" {
		destCaller = testutil.RandomBytes(32)
	} else if destCallerStr != "" {
		destCaller, err = decodeHexOrBase64To32Bytes(destCallerStr)
		if err != nil {
			m.err = fmt.Errorf("invalid destination caller: %w", err)

			return m, nil
		}
	}

	var passthroughPayload []byte
	if passthroughStr != "" {
		passthroughPayload = []byte(passthroughStr)
	}

	cctpForwarding, err := forwarding.NewCCTPForwarding(
		uint32(domain),
		mintRecipient,
		destCaller,
		passthroughPayload,
	)
	if err != nil {
		m.err = fmt.Errorf("failed to create CCTP forwarding: %w", err)

		return m, nil
	}

	m.forwarding = cctpForwarding

	m.payload, err = buildFinalPayload(m.forwarding, m.actions)
	if err != nil {
		m.err = fmt.Errorf("failed to build finalPayload: %w", err)

		return m, nil
	}

	return m, tea.Quit
}

func (m Model) updateForwardingInputs(msg tea.Msg) tea.Cmd {
	if len(m.forwardingInputs) == 0 {
		return nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case Tab, ShiftTab, Up, Down:
			s := msg.String()

			// Update focus position
			switch s {
			case Up, ShiftTab:
				if focusIndex > 0 {
					focusIndex--
				}
			case Down, Tab:
				if focusIndex < len(m.forwardingInputs)-1 {
					focusIndex++
				}
			}

			// Update focus for all inputs
			cmds := make([]tea.Cmd, len(m.forwardingInputs))
			for i := range m.forwardingInputs {
				if i == focusIndex {
					cmds[i] = m.forwardingInputs[i].Focus()
				} else {
					m.forwardingInputs[i].Blur()
				}
			}

			return tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking for all inputs
	cmds := make([]tea.Cmd, len(m.forwardingInputs))
	for i := range m.forwardingInputs {
		m.forwardingInputs[i], cmds[i] = m.forwardingInputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

// decodeHexOrBase64To32Bytes decodes a string as either a hex or base64 encoded string.
// It returns a 32 byte slice, or an error if the input is invalid.
func decodeHexOrBase64To32Bytes(input string) (decoded []byte, err error) {
	if strings.HasPrefix(input, "0x") {
		decoded, err = hexutil.Decode(input)
		if err != nil {
			return nil, fmt.Errorf("failed to decode hex: %w", err)
		}
	} else {
		decoded, err = base64.StdEncoding.DecodeString(input)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
	}

	return leftPadIfRequired(decoded)
}

// leftPadIfRequired pads a byte slice to the left with 0x00 if the length is not 32 bytes.
func leftPadIfRequired(input []byte) ([]byte, error) {
	inputLen := len(input)
	if inputLen > 32 {
		return nil, fmt.Errorf("input is too long; max 32 bytes; got: %d", inputLen)
	}

	if inputLen == 32 {
		return input, nil
	}

	pad := make([]byte, 32-inputLen)

	return append(pad, input...), nil
}
