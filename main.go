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

package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/noble-assets/orbiter/testutil"

	"github.com/noble-assets/orbgen/internal"
)

func main() {
	// NOTE: this is required to be called to correctly set the bech32 prefix
	testutil.SetSDKConfig()

	// Setup the TUI model and run it
	m := internal.InitialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	runModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Print the full payload to stdout when exiting
	//
	// NOTE: This is not handled within the charm stuff to enable copying the full thing.
	// Within the charm TUI, the output would be truncated to the size of the window.
	if runModel != nil {
		m, ok := runModel.(internal.Model)
		if !ok {
			log.Fatal(fmt.Errorf("unexpected model; got %T", runModel))
		}

		fmt.Println(m.GetPayload())
	}
}
