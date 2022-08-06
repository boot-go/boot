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
)

// registry contains all created component componentManagers.
type registry struct {
	// items are organized in hierarchy, using the component name, componentManager name and containing
	// the componentManager.
	items map[string]map[string]*componentManager
	// executionWaitGroup tracks the amount off active components
	executionWaitGroup sync.WaitGroup
}

// newRegistry creates a new component registry.
func newRegistry() *registry {
	return &registry{
		items:              make(map[string]map[string]*componentManager),
		executionWaitGroup: sync.WaitGroup{},
	}
}

// addItem adds a component componentManager to the registry.
func (reg *registry) addItem(name string, override bool, cmp Component) error {
	cmpMngr := newComponentManager(name, cmp, &reg.executionWaitGroup)
	id := cmpMngr.getName()
	if reg.items[id] == nil {
		// enter first componentManager in registry
		v := make(map[string]*componentManager)
		v[name] = cmpMngr
		reg.items[id] = v
		Logger.Debug.Printf("creating %s", cmpMngr.getFullName())
	} else {
		registeredComponent := reg.items[id][name]
		if registeredComponent == nil {
			// items already found, but not for given name
			reg.items[id][name] = cmpMngr
			Logger.Debug.Printf("creating %s\n", cmpMngr.getFullName())
		} else {
			if override {
				Logger.Debug.Printf("overriding %s\n", cmpMngr.getFullName())
				reg.items[id][name] = cmpMngr
			} else {
				// a component exists with the given name
				return errors.New("go aborted because component " + id + " already registered under the name '" + name + "'")
			}
		}
	}
	return nil
}

func (reg *registry) resolveComponentDependencies() (componentManagers, error) {
	var entries []*componentManager
	for _, cmpTypList := range reg.items {
		for _, entry := range cmpTypList {
			newEntries, err := resolveDependency(entry, reg)
			if err != nil {
				return nil, err
			}
			entries = append(entries, newEntries...)
		}
	}
	return entries, nil
}
