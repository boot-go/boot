/*
 * Copyright (c) 2021-2023 boot-go
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

import "sync"

// componentManager represents a registry entity containing the component with its metadata.
type componentManager struct {
	// component is the running global componentManager.
	component Component
	// state contains the component state.
	state componentState
	// name is used to identify the component.
	name string
	// stateChangeMutex to prevent race conditions
	stateChangeMutex *sync.Mutex
	// waitGroup is used to block the main process until all Processes are stopped
	waitGroup *sync.WaitGroup
}

// componentState is used to describe the current state of a component componentManager
type componentState int

const (
	// Created is set directly after the component was successfully created by the provided factory
	Created componentState = iota
	// Initialized is set after the component Init() function was called
	Initialized
	// Started is set after the component Start() function was called
	Started
	// Stopped is set after the component Stop() function was called
	Stopped
	// Failed is set when the component couldn't be initialized
	Failed
)

func newComponentManager(name string, cmp Component, wg *sync.WaitGroup) *componentManager {
	return &componentManager{
		name:             name,
		component:        cmp,
		state:            Created,
		stateChangeMutex: &sync.Mutex{},
		waitGroup:        wg,
	}
}

// getFullName() return the componentManager name with name of the component separated by a colon.
// E.g. default:github.com/boot-go/boot/boot/runtime
func (cm *componentManager) getFullName() string {
	return cm.name + ":" + QualifiedName(cm.component)
}

// getName returns the qualified name
func (cm *componentManager) getName() string {
	return QualifiedName(cm.component)
}

// start will call the start function inside Component, if it is not nil
func (cm *componentManager) start() {
	if process, ok := cm.component.(Process); ok {
		cm.stateChangeMutex.Lock()
		if cm.state == Initialized {
			cm.waitGroup.Add(1)
			go func() {
				cm.stateChangeMutex.Lock()
				cm.state = Started
				cm.stateChangeMutex.Unlock()
				Logger.Debug.Printf("starting %s", cm.getFullName())
				err := process.Start()
				cm.stateChangeMutex.Lock()
				if cm.state == Started {
					cm.waitGroup.Done()
					if err == nil {
						cm.state = Stopped
					} else {
						cm.state = Failed
						Logger.Error.Printf("process.Start() failed: %v", err)
					}
				}
				cm.stateChangeMutex.Unlock()
			}()
		}
		cm.stateChangeMutex.Unlock()
	}
}

// stop will call the stop function inside Component, if it is not nil
func (cm *componentManager) stop() {
	if process, ok := cm.component.(Process); ok {
		cm.stateChangeMutex.Lock()
		if cm.state == Started {
			Logger.Debug.Printf("stopping %s", cm.getFullName())
			err := process.Stop()
			if err != nil {
				Logger.Error.Printf("process.Stop() failed: %v", err)
			}
			cm.state = Stopped
			cm.waitGroup.Done()
		}
		cm.stateChangeMutex.Unlock()
	}
}

type componentManagers []*componentManager

func (e componentManagers) stopComponents() {
	for i := range e {
		e := e[len(e)-i-1]
		e.stop()
	}
}

func (e componentManagers) startComponents() {
	for _, e := range e {
		e.start()
	}
}

func (e componentManagers) count() int {
	return len(e)
}
