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

// phase describes the status of the boot-go componentManager
type phase uint8

// String returns the name of the phase
func (p phase) String() string {
	switch p {
	case initializing:
		return "initialization"
	case booting:
		return "booting"
	case running:
		return "running"
	case stopping:
		return "stopping"
	case exiting:
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
	// initializing is set directly after the application started.
	// In this phase it is safe to subscribe to events.
	initializing phase = iota
	// booting is set when starting the boot framework.
	booting
	// running is set when all components were initialized and started
	running
	// stopping is set when all components are requested to stop their service.
	stopping
	// exiting is set when all components were stopped.
	exiting
)

// Session is the main struct for the boot-go application framework
type Session struct {
	factories   []factory
	changeMutex sync.Mutex
	phase       phase
	runtime     *runtime
	eventbus    *eventBus
	option      Options
}

// Options contains the options for the boot-go Session
type Options struct {
	// Mode is a list of flags to be used for the application.
	Mode []Flag
	// DoMain is called when the application is requested to start and blocks until shutdown is requested.
	DoMain func() error
	// DoShutdown is called when the application is requested to shutdown.
	DoShutdown func() error
	// channel to receive shutdown or interrupt signal - this is used for testing
	shutdownChannel chan os.Signal
}

// NewSession will create a new Session with default options
func NewSession(mode ...Flag) *Session {
	localShutdownChannel := make(chan os.Signal, 1)
	return NewSessionWithOptions(Options{
		Mode: mode,
		DoMain: func() error {
			signal.Notify(localShutdownChannel, interruptSignal, shutdownSignal)
			sig := <-localShutdownChannel
			switch {
			case sig == interruptSignal:
				Logger.Warn.Printf("caught interrupt signal %s\n", sig.String())
				Logger.Debug.Printf("shutdown gracefully initiated...\n")
			case sig == shutdownSignal:
				Logger.Debug.Printf("shutdown requested...\n")
			}
			return nil
		},
		DoShutdown: func() error {
			localShutdownChannel <- shutdownSignal
			return nil
		},
		shutdownChannel: localShutdownChannel,
	})
}

// NewSessionWithOptions will create a new Session with given options
func NewSessionWithOptions(options Options) *Session {
	s := &Session{
		factories:   []factory{},
		changeMutex: sync.Mutex{},
		phase:       initializing,
		option:      options,
	}
	// register default components... errors not possible, so they are ignored
	s.runtime = &runtime{
		modes: options.Mode,
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

// nextPhaseAfter will change the current phase to the next phase. If the current phase is not the expected phase, an error will be returned.
func (s *Session) nextPhaseAfter(expected phase) error {
	defer s.changeMutex.Unlock()
	s.changeMutex.Lock()
	newPhase := initializing
	switch s.phase {
	case initializing:
		newPhase = booting
	case booting:
		newPhase = running
	case running:
		newPhase = stopping
	case stopping:
		newPhase = exiting
	case exiting:
		// there is no new phase, because it would be exited
	}
	if expected != s.phase {
		return errors.New("current boot phase " + s.phase.String() + " doesn't match expected boot phase " + expected.String())
	}
	Logger.Debug.Printf("boot phase changed from " + s.phase.String() + " to " + newPhase.String())
	s.phase = newPhase
	return nil
}

// register a factory function for a component. These functions will be called on boot to create the components.
func (s *Session) register(name string, create func() Component, override bool) error {
	if name == "" || create == nil {
		return errSessionRegisterNameOrFunction
	}
	defer s.changeMutex.Unlock()
	s.changeMutex.Lock()
	if s.phase != initializing {
		return errSessionRegisterComponentOutsideInitialize
	}
	s.factories = append(s.factories, factory{
		create:   create,
		name:     name,
		override: override,
	})
	return nil
}

// Register a factory function for a component. The component will be created on boot.
func (s *Session) Register(create func() Component) error {
	return s.register(DefaultName, create, false)
}

// Override a factory function for a component. The component will be created on boot.
func (s *Session) Override(create func() Component) error {
	return s.register(DefaultName, create, true)
}

// RegisterName registers a factory function with the given name. The component will be created on boot.
func (s *Session) RegisterName(name string, create func() Component) error {
	return s.register(name, create, false)
}

// OverrideName overrides a factory function with the given name. The component will be created on boot.
func (s *Session) OverrideName(name string, create func() Component) error {
	return s.register(name, create, true)
}

// Go the boot component framework. This starts the execution process.
func (s *Session) Go() error { //nolint:varnamelen // s is fine for method
	if err := s.nextPhaseAfter(initializing); err != nil {
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

	if err := s.nextPhaseAfter(booting); err != nil {
		return err
	}
	instances.startComponents()
	Logger.Debug.Printf("%d components started", instances.count())

	go func() {
		err := s.waitUntilAllComponentsStopped(registry)
		if err != nil {
			Logger.Error.Printf("shutdown failed with error: %v", err)
		}
	}()

	// activate eventbus to process alle queued events
	err = s.eventbus.activate()
	if err == nil {
		// blocking here until Shutdown
		err = s.option.DoMain()
		if err != nil {
			Logger.Error.Printf("processing until shutdown failed with error: %v", err)
			return err
		}
	} else {
		Logger.Error.Printf("going down - eventbus activation failed: %v", err)
	}

	if err := s.nextPhaseAfter(running); err != nil {
		Logger.Error.Printf("component stop error: %v", err)
	}
	instances.stopComponents()
	Logger.Debug.Printf("%d components stopped", instances.count())

	if err := s.nextPhaseAfter(stopping); err != nil {
		return err
	}

	Logger.Debug.Printf("boot done")
	return nil
}

// Shutdown initiates the shutdown process. All components will be stopped.
func (s *Session) Shutdown() error {
	Logger.Debug.Printf("shutdown initiated...")
	if s.option.DoShutdown != nil {
		err := s.option.DoShutdown()
		if err != nil {
			return err
		}
	}
	return nil
}

// createComponents() will create all registered components
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

// waitUntilAllComponentsStopped() will wait until all components have stopped processing
func (s *Session) waitUntilAllComponentsStopped(reg *registry) error {
	Logger.Debug.Printf("wait until all components are stopped...")
	reg.executionWaitGroup.Wait()
	err := s.Shutdown()
	if err != nil {
		return err
	}
	return err
}
