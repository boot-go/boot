/*
 * Copyright (c) 2021-2022 boot-go
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 */

package boot

import (
	"errors"
	"sync"
	"testing"
)

type componentManagerTest struct{}

func (c *componentManagerTest) Init() error {
	return nil
}

func (c *componentManagerTest) Start() error {
	return errors.New("fail")
}

func (c *componentManagerTest) Stop() error {
	return errors.New("fail")
}

func TestComponentManagerStart(t *testing.T) {
	type fields struct {
		component        Component
		state            componentState
		name             string
		stateChangeMutex *sync.Mutex
		waitGroup        *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{name: "with error", fields: struct {
			component        Component
			state            componentState
			name             string
			stateChangeMutex *sync.Mutex
			waitGroup        *sync.WaitGroup
		}{component: &componentManagerTest{}, state: Initialized, name: DefaultName, stateChangeMutex: &sync.Mutex{}, waitGroup: &sync.WaitGroup{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &componentManager{
				component:        tt.fields.component,
				state:            tt.fields.state,
				name:             tt.fields.name,
				stateChangeMutex: tt.fields.stateChangeMutex,
				waitGroup:        tt.fields.waitGroup,
			}
			e.start()
		})
	}
}

func TestComponentManagerStop(t *testing.T) {
	type fields struct {
		component        Component
		state            componentState
		name             string
		stateChangeMutex *sync.Mutex
		waitGroup        *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{name: "with error", fields: struct {
			component        Component
			state            componentState
			name             string
			stateChangeMutex *sync.Mutex
			waitGroup        *sync.WaitGroup
		}{component: &componentManagerTest{}, state: Started, name: DefaultName, stateChangeMutex: &sync.Mutex{}, waitGroup: &sync.WaitGroup{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &componentManager{
				component:        tt.fields.component,
				state:            tt.fields.state,
				name:             tt.fields.name,
				stateChangeMutex: tt.fields.stateChangeMutex,
				waitGroup:        tt.fields.waitGroup,
			}
			e.waitGroup.Add(1)
			e.stop()
		})
	}
}
