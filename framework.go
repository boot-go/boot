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
	"fmt"
	"os"
	"os/signal"
	gort "runtime"
	"sync"
	"syscall"
	"time"
)

// Component represent functional building blocks, which solve one specific purpose.
// They should be fail tolerant, recoverable, agnostic and decent.
//
// fail tolerant: Don't stop processing on errors.
// Example: A http request can still be processed, even when the logging server is not available
//          anymore.
//
// recoverable: Try to recover from errors.
// Example: A database component should try to reconnect after lost connection.
//
// agnostic: Behave the same in any environment.
// Example: A key-value store component should work on a local development machine the same way as
//          in a containerized environment.
//
// decent: Don't overload the developer with complexity.
// Example: Keep the interface and events as simple as possible. Less is often more.
type Component interface {
	// Init initializes data, set the default configuration, subscribe to events
	//or performs other kind of configuration.
	Init()
}

// Process is a Component which has a processing functionality. This can be anything like a server,
// cron job or long running process.
type Process interface {
	Component
	// Start is called as soon as all boot.Component components are initialized. The call should be
	// blocking until all processing is completed.
	Start()
	// Stop is called to abort the processing and clean up resources. Pay attention that the
	// processing may already be stopped.
	Stop()
}

// factory contains a name, some metadata and factory function for a given component.
type factory struct {
	create   func() Component
	name     string
	override bool
}

const (
	// DefaultName is used when registering components without an explicit name.
	DefaultName = "default"
	// shutdownSignal uses SIGTERM. The SIGTERM signal is sent to a process to request its
	//termination. It will also be used when Shutdown() is called.
	shutdownSignal = syscall.SIGTERM
)

// Phase describes the status of the boot-go instance
type Phase uint8

func (p Phase) String() string {
	switch p {
	case Initializing:
		return "initialization"
	case Booting:
		return "booting"
	case Running:
		return "running"
	case Stopping:
		return "stopping"
	case Exiting:
		return "exiting"
	}
	return "unknown"
}

const (
	// Initializing is set directly after the application started.
	Initializing Phase = iota
	// Booting is set when starting the boot framework.
	Booting
	// Running is set when all components were initialized and started
	Running
	// Stopping is set when all components are requested to stop their service.
	Stopping
	// Exiting is set when all components were stopped.
	Exiting
)

var (
	factories       []factory
	phaseMutex      sync.Mutex
	phase           Phase
	shutdownChannel chan os.Signal
)

func init() {
	setup(StandardFlag)
}

func setup(mode Flag) {
	// initialize global values
	factories = []factory{}
	phase = Initializing
	phaseMutex = sync.Mutex{}
	// register default components
	Register(func() Component {
		return newEventbus()
	})
	Register(func() Component {
		return &runtime{
			modes: []Flag{mode},
		}
	})
}

func nextPhaseAfter(expected Phase) error {
	newPhase := Initializing
	switch phase {
	case Initializing:
		newPhase = Booting
	case Booting:
		newPhase = Running
	case Running:
		newPhase = Stopping
	case Stopping:
		newPhase = Exiting
	}
	defer phaseMutex.Unlock()
	phaseMutex.Lock()
	if expected != phase {
		return errors.New("current boot phase " + phase.String() + " doesn't match expected boot phase " + expected.String())
	}
	Logger.Debug.Printf("boot phase changed from " + phase.String() + " to " + newPhase.String())
	phase = newPhase
	return nil
}

// Register a default factory function.
func Register(create func() Component) {
	RegisterName(DefaultName, create)
}

// RegisterName registers a factory function with the given name.
func RegisterName(name string, create func() Component) {
	if name == "" || create == nil {
		panic("Name for component factory registration is required!")
	}
	if phase != Initializing {
		panic("Registering Component not allowed after boot has been started!")
	}
	factories = register(factories, name, create)
}

func register(factoryList []factory, name string, create func() Component) []factory {
	return append(factories, factory{
		create:   create,
		name:     name,
		override: false,
	})
}

// Override a default factory function.
func Override(create func() Component) {
	OverrideName(DefaultName, create)
}

// OverrideName overrides a factory function with the given name.
func OverrideName(name string, create func() Component) {
	factories = override(factories, name, create)
}

func override(factoryList []factory, name string, create func() Component) []factory {
	if name == "" || create == nil {
		panic("Name for component factory override registration is required!")
	}
	if phase != Initializing {
		panic("Overriding component not allowed after boot has been started!")
	}
	factoryList = append(factoryList, factory{
		create:   create,
		name:     name,
		override: true,
	})
	return factoryList
}

// Go the boot component framework. This starts the execution process.
func Go() error {
	startTime := time.Now()
	s := new(gort.MemStats)
	gort.ReadMemStats(s)
	// output some basic info
	Logger.Info.Printf("booting `boot-go %s` /// %s OS/%s ARCH/%s CPU/%d MEM/%dMB SYS/%dMB\n", version, gort.Version(), gort.GOOS, gort.GOARCH, gort.NumCPU(), (s.Alloc / 1024 / 1024), (s.Sys / 1024 / 1024))
	entries, err := run(factories)
	Logger.Debug.Printf("boot done with %d components", len(entries))
	if err == nil {
		Logger.Info.Printf("shutdown after %s\n", time.Now().Sub(startTime).String())
	} else {
		Logger.Error.Printf("shutdown after %s with: %s\n", time.Now().Sub(startTime).String(), err.Error())
	}
	return err
}

func run(factoryList []factory) ([]*entry, error) {
	if err := nextPhaseAfter(Initializing); err != nil {
		return nil, err
	}

	registry, err := createComponents(factoryList)
	if err != nil {
		return nil, err
	}
	entries, err := resolveComponentDependencies(registry)
	if err != nil {
		return nil, err
	}
	shutdownHandler(entries)
	if err = startComponents(entries); err != nil {
		return entries, err
	}
	registry.waitUntilAllComponentsStopped()
	if phase == Running {
		_ = stopComponents(entries)
	}
	if err := nextPhaseAfter(Stopping); err != nil {
		return entries, err
	}
	return entries, nil
}

func shutdownHandler(entries []*entry) {
	shutdownChannel = make(chan os.Signal, 1)
	go func() {
		signal.Notify(shutdownChannel, syscall.SIGINT, syscall.SIGKILL, shutdownSignal)
		sig := <-shutdownChannel
		switch {
		case sig == syscall.SIGINT || sig == syscall.SIGKILL:
			Logger.Warn.Printf("caught signal: %s\n", sig.String())
			Logger.Debug.Printf("shutdown gracefully initiated...\n")
		case sig == shutdownSignal:
			Logger.Debug.Printf("shutdown requested...\n")
		}
		_ = stopComponents(entries)
		Logger.Info.Printf("shutdown completed\n")
	}()
}

// Shutdown boot-go instance. All components will be stopped. This is equivalent with
// issuing a SIGTERM on process level.
func Shutdown() {
	shutdownChannel <- shutdownSignal
}

func createComponents(factoryList []factory) (*registry, error) {
	registry := newRegistry()
	for _, factory := range factoryList {
		component := factory.create()
		if component == nil {
			return nil, fmt.Errorf("factory %s failed to create a component", QualifiedName(factory))
		}
		err := registry.addEntry(factory.name, factory.override, component)
		if err != nil {
			return registry, err
		}
	}
	return registry, nil
}

func stopComponents(entries []*entry) error {
	if err := nextPhaseAfter(Running); err != nil {
		Logger.Error.Printf("stopping failed with %s\n", err)
		return err
	}
	for i := range entries {
		e := entries[len(entries)-i-1]
		e.stop()
	}
	return nil
}

func startComponents(entries []*entry) error {
	if err := nextPhaseAfter(Booting); err != nil {
		return err
	}
	for _, e := range entries {
		e.start()
	}
	return nil
}

func resolveComponentDependencies(registry *registry) ([]*entry, error) {
	var entries []*entry
	for _, cmpTypList := range registry.entries {
		for _, entry := range cmpTypList {
			newEntries, err := resolveDependency(entry, registry)
			if err != nil {
				return nil, err
			}
			entries = append(entries, newEntries...)
		}
	}
	return entries, nil
}
