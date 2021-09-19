/*
 * Copyright (c) 2021 boot-go
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
)

// registry contains all created component instances.
type registry struct {
	// entries are organized in hierarchy, using the component name, instance name and containing
	//the entry.
	entries map[string]map[string]*entry
	// executionWaitGroup tracks the amount off active components
	executionWaitGroup sync.WaitGroup
}

// entry represents a registry entity containing the component with its meta data.
type entry struct {
	// component is the running global instance.
	component Component
	// state contains the component state.
	state State
	// name is used to identify the component.
	name string
	// stateChangeMutex to prevent race conditions
	stateChangeMutex sync.Mutex
	// registry is the reference where the entry has been stored
	registry *registry
}

// State is used to describe the current state of a component instance
type State int

const (
	// Created is set directly after the component was created by the provided factory
	Created State = iota
	// Initialized is set after the component Init() function was called
	Initialized
	// Started is set after the component Start() function was called
	Started
	// Stopped is set after the component Stop() function was called
	Stopped
)

// newRegistry creates a new component registry.
func newRegistry() *registry {
	return &registry{
		entries:            make(map[string]map[string]*entry),
		executionWaitGroup: sync.WaitGroup{},
	}
}

// addEntry adds a component instance to the registry.
func (r *registry) addEntry(name string, override bool, cmp Component) error {
	e := &entry{
		name:      name,
		component: cmp,
		state:     Created,
		registry:  r,
	}
	id := QualifiedName(cmp)
	if r.entries[id] == nil {
		// enter first entry in registry
		v := make(map[string]*entry)
		v[name] = e
		r.entries[id] = v
		Logger.Debug.Printf("creating %s", e.getFullName())
	} else {
		registeredComponent := r.entries[id][name]
		if registeredComponent == nil {
			// entries already found, but not for given name
			r.entries[id][name] = e
			Logger.Debug.Printf("creating %s\n", e.getFullName())
		} else {
			if override {
				Logger.Debug.Printf("overriding %s\n", e.getFullName())
				r.entries[id][name] = e
			} else {
				// a component exists with the given name
				return errors.New("go aborted because component " + id + " already registered under the name '" + name + "'")
			}
		}
	}
	return nil
}

// waitUntilAllComponentsStopped() will wait until all components have stopped processing
func (r *registry) waitUntilAllComponentsStopped() {
	Logger.Debug.Printf("wait until all components are stopped...")
	r.executionWaitGroup.Wait()
	Logger.Debug.Printf("all components stopped")
}

// getFullName() return the instance name with name of the component separated by a colon.
//E.g. default:github.com/boot-go/boot/boot/runtime
func (e *entry) getFullName() string {
	return e.name + ":" + QualifiedName(e.component)
}

// start will call the start function inside Component, if it is not nil
func (e *entry) start() {
	if runnable, ok := e.component.(Process); ok {
		e.stateChangeMutex.Lock()
		if e.state == Initialized {
		e.registry.executionWaitGroup.Add(1)
			go func() {
				Logger.Debug.Printf("starting %s", e.getFullName())
				runnable.Start()
				e.stateChangeMutex.Lock()
				if e.state == Started {
					e.state = Stopped
					e.registry.executionWaitGroup.Done()
				}
				e.stateChangeMutex.Unlock()
			}()
			e.state = Started
		}
		e.stateChangeMutex.Unlock()
	}
}

// stop will call the stop function inside Component, if it is not nil
func (e *entry) stop() {
	if process, ok := e.component.(Process); ok {
		e.stateChangeMutex.Lock()
		if e.state == Started {
			Logger.Debug.Printf("stopping %s", e.getFullName())
			process.Stop()
			e.state = Stopped
			e.registry.executionWaitGroup.Done()
		}
		e.stateChangeMutex.Unlock()
	}
}
