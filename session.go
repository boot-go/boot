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
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// factory contains a name, some metadata and factory function for a given component.
type factory struct {
	create   func() Component
	name     string
	override bool
}

// Phase describes the status of the boot-go componentManager
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
	// shutdownSignal uses SIGTERM. The SIGTERM signal is sent to a process to request its
	// termination. It will also be used when Shutdown() is called.
	shutdownSignal = syscall.SIGTERM
	// interruptSignal will be used by the Session.
	interruptSignal = syscall.SIGINT
)

var (
	errSessionRegisterNameOrFunction             = errors.New("name and function for component factory registration is required")
	errSessionRegisterComponentOutsideInitialize = errors.New("register component not allowed after boot has been started")
)

const (
	// Initializing is set directly after the application started.
	// In this phase it is safe to subscribe to events.
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

type Session struct {
	factories       []factory
	changeMutex     sync.Mutex
	phase           Phase
	shutdownChannel chan os.Signal
	runtime         *runtime
	eventbus        *eventBus
}

// NewSession will create a new Session
func NewSession(mode ...Flag) *Session {
	s := &Session{
		factories:       []factory{},
		changeMutex:     sync.Mutex{},
		phase:           Initializing,
		shutdownChannel: make(chan os.Signal, 1),
	}
	// register default components... errors not possible, so they are ignored
	s.runtime = &runtime{
		modes: mode,
	}
	_ = s.register(DefaultName, func() Component {
		return s.runtime
	}, false)
	s.eventbus = newEventbus()
	_ = s.register(DefaultName, func() Component {
		return s.eventbus
	}, false)
	return s
}

func (s *Session) nextPhaseAfter(expected Phase) error {
	defer s.changeMutex.Unlock()
	s.changeMutex.Lock()
	newPhase := Initializing
	switch s.phase {
	case Initializing:
		newPhase = Booting
	case Booting:
		newPhase = Running
	case Running:
		newPhase = Stopping
	case Stopping:
		newPhase = Exiting
	case Exiting:
		// there is no new phase, because it would be exited
	}
	if expected != s.phase {
		return errors.New("current boot phase " + s.phase.String() + " doesn't match expected boot phase " + expected.String())
	}
	Logger.Debug.Printf("boot phase changed from " + s.phase.String() + " to " + newPhase.String())
	s.phase = newPhase
	return nil
}

func (s *Session) register(name string, create func() Component, override bool) error {
	if name == "" || create == nil {
		return errSessionRegisterNameOrFunction
	}
	defer s.changeMutex.Unlock()
	s.changeMutex.Lock()
	if s.phase != Initializing {
		return errSessionRegisterComponentOutsideInitialize
	}
	s.factories = append(s.factories, factory{
		create:   create,
		name:     name,
		override: override,
	})
	return nil
}

func (s *Session) Register(create func() Component) error {
	return s.register(DefaultName, create, false)
}

func (s *Session) Override(create func() Component) error {
	return s.register(DefaultName, create, true)
}

func (s *Session) RegisterName(name string, create func() Component) error {
	return s.register(name, create, false)
}

func (s *Session) OverrideName(name string, create func() Component) error {
	return s.register(name, create, true)
}

func (s *Session) Go() error { //nolint:varnamelen // s is fine for method
	if err := s.nextPhaseAfter(Initializing); err != nil {
		return err
	}

	registry, err := s.createComponents()
	if err != nil {
		return err
	}
	instances, err := registry.resolveComponentDependencies()
	if err != nil {
		return err
	}

	if err := s.nextPhaseAfter(Booting); err != nil {
		return err
	}
	instances.startComponents()
	Logger.Debug.Printf("%d components started", instances.count())

	go s.waitUntilAllComponentsStopped(registry)

	// activate eventbus to process alle queued events
	err = s.eventbus.activate()
	if err == nil {
		// blocking here until Shutdown
		s.waitForShutdown(instances)
	} else {
		Logger.Error.Printf("going down - eventbus activation failed: %v", err)
	}

	if err := s.nextPhaseAfter(Running); err != nil {
		Logger.Error.Printf("component stop error: %v", err)
	}
	instances.stopComponents()
	Logger.Debug.Printf("%d components stopped", instances.count())

	if err := s.nextPhaseAfter(Stopping); err != nil {
		return err
	}

	Logger.Debug.Printf("boot done")
	return nil
}

func (s *Session) Shutdown() {
	s.shutdownChannel <- shutdownSignal
}

func (s *Session) createComponents() (*registry, error) {
	registry := newRegistry()
	for _, factory := range s.factories {
		component := factory.create()
		if component == nil {
			return nil, fmt.Errorf("factory %s failed to create a component", QualifiedName(factory))
		}
		err := registry.addItem(factory.name, factory.override, component)
		if err != nil {
			return registry, err
		}
	}
	return registry, nil
}

func (s *Session) waitForShutdown(instances componentManagers) {
	signal.Notify(s.shutdownChannel, interruptSignal, shutdownSignal)
	sig := <-s.shutdownChannel
	switch {
	case sig == interruptSignal:
		Logger.Warn.Printf("caught interrupt signal %s\n", sig.String())
		Logger.Debug.Printf("Shutdown gracefully initiated...\n")
	case sig == shutdownSignal:
		Logger.Debug.Printf("Shutdown requested...\n")
	}
}

// waitUntilAllComponentsStopped() will wait until all components have stopped processing
func (s *Session) waitUntilAllComponentsStopped(reg *registry) {
	Logger.Debug.Printf("wait until all components are stopped...")
	reg.executionWaitGroup.Wait()
	s.Shutdown()
}
