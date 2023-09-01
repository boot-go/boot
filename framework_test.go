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
	"math"
	"sync"
	"testing"
	"time"
)

type bootTestComponent struct {
	initCalled  bool
	startCalled bool
	stopCalled  bool
}

func (t *bootTestComponent) Init() error {
	t.initCalled = true
	return nil
}

func (t *bootTestComponent) Start() error {
	t.startCalled = true
	return nil
}

func (t *bootTestComponent) Stop() error {
	t.stopCalled = true
	return nil
}

type bootProcessesComponent struct {
	block   chan bool
	stopped bool
	mutex   sync.Mutex
}

func (t *bootProcessesComponent) Init() error {
	t.block = make(chan bool, 1)
	t.mutex = sync.Mutex{}
	defer t.mutex.Unlock()
	t.mutex.Lock()
	t.stopped = false
	return nil
}

func (t *bootProcessesComponent) Start() error {
	<-t.block
	defer t.mutex.Unlock()
	t.mutex.Lock()
	t.stopped = true
	return nil
}

func (t *bootProcessesComponent) Stop() error {
	t.block <- true
	close(t.block)
	return nil
}

type bootMissingDependencyComponent struct {
	WireFails string `boot:"wire"`
}

func (t *bootMissingDependencyComponent) Init() error {
	return nil
}

type bootPanicComponent struct {
	content any
}

func (t *bootPanicComponent) Init() error {
	panic(t.content)
}

func TestBootGo(t *testing.T) {
	testStruct := &bootTestComponent{}
	ts := newTestSession(testStruct)

	err := ts.Go()
	if err != nil {
		t.FailNow()
	}

	if !testStruct.initCalled ||
		!testStruct.startCalled ||
		testStruct.stopCalled {
		t.Fail()
	}
}

func TestBootAlreadyRegisteredComponent(t *testing.T) {
	testStruct := &bootTestComponent{}
	ts := newTestSession()
	err := ts.registerTestComponent(testStruct, testStruct)
	if err != nil {
		t.Failed()
	}

	err = ts.Go()
	if err != nil && err.Error() == "go aborted because component github.com/boot-go/boot/bootTestComponent already registered under the name 'default'" {
		return
	}
	t.Fatal("error expected on already registered component")
}

func TestBootFactoryFail(t *testing.T) {
	err := newTestSession()
	if err == nil {
		t.FailNow()
	}
}

func TestBootWithErrorComponent(t *testing.T) {
	tests := []struct {
		name    string
		content any
		err     error
	}{
		{name: "string content", content: "string content", err: errors.New("initializing default:github.com/boot-go/boot/bootPanicComponent panicked with message: string content")},
		{name: "error content", content: errors.New("error content"), err: errors.New("initializing default:github.com/boot-go/boot/bootPanicComponent panicked with error: error content")},
		{name: "other content", content: 0, err: errors.New("initializing default:github.com/boot-go/boot/bootPanicComponent panicked")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testStruct := &bootPanicComponent{content: tt.content}
			ts := newTestSession()
			err := ts.overrideTestComponent(testStruct)
			if err != nil {
				t.Failed()
			}
			err = ts.Go()
			if err == nil || err.Error() != tt.err.Error() {
				t.Errorf("Expected '%s' but found '%s'", tt.err.Error(), err.Error())
			}
		})
	}
}

func TestBootShutdown(t *testing.T) {
	testStruct := &bootProcessesComponent{}
	ts := newTestSession(testStruct)

	go func() {
		time.Sleep(5 * time.Second)
		err := ts.Shutdown()
		if err != nil {
			t.Fail()
		}
	}()

	err := ts.Go()
	if err != nil {
		t.FailNow()
	}

	time.Sleep(2 * time.Second)
	defer testStruct.mutex.Unlock()
	testStruct.mutex.Lock()
	if !testStruct.stopped {
		t.Fatal("Component not stopped")
	}
}

func TestBootShutdownFails(t *testing.T) {
	globalSession = NewSession(UnitTestFlag)
	Register(func() Component {
		return &bootTestComponent{}
	})
	globalSession.option.DoShutdown = func() error {
		globalSession.option.shutdownChannel <- shutdownSignal
		return errors.New("shutdown failed")
	}

	testSucceeded := false
	mutex := sync.Mutex{}

	go func() {
		time.Sleep(2 * time.Second)
		err := Shutdown()
		if err == nil || err.Error() != "shutdown failed" {
			t.Fail()
		} else {
			mutex.Lock()
			testSucceeded = true
			mutex.Unlock()
		}
	}()

	err := Go()
	if err != nil {
		t.FailNow()
	}

	time.Sleep(5 * time.Second)
	defer mutex.Unlock()
	mutex.Lock()
	if !testSucceeded {
		t.Fatal("shutdown test failed")
	}
}

func TestBootAlreadyRunningThenRegister(t *testing.T) {
	testStruct := &bootTestComponent{}
	ts := newTestSession()
	err := ts.registerTestComponent(testStruct)
	if err != nil {
		t.Fatal("first registration should not fail")
	}

	err = ts.Go()
	if err != nil {
		t.Fatal("boot should not fail to start")
	}
	err = ts.registerTestComponent(testStruct)
	if err == nil {
		t.Fatal("error expected on boot started but registering component")
	}
}

func TestBootAlreadyRunningThenOverride(t *testing.T) {
	testStruct := &bootTestComponent{}
	ts := newTestSession()
	err := ts.registerTestComponent(testStruct)
	if err != nil {
		t.Fatal("first registration should not fail")
	}

	err = ts.Go()
	if err != nil {
		t.Fatal("boot should not fail to start")
	}
	err = ts.overrideTestComponent(testStruct)
	if err == nil {
		t.Fatal("error expected on boot started but overriding component")
	}
}

func TestShutdownByOsSignal(t *testing.T) {
	testStruct := &bootProcessesComponent{}
	ts := newTestSession(testStruct)

	go func() {
		time.Sleep(2 * time.Second)
		ts.Session.option.shutdownChannel <- shutdownSignal
	}()

	err := ts.Go()
	if err != nil {
		t.FailNow()
	}

	time.Sleep(1 * time.Second)
	defer testStruct.mutex.Unlock()
	testStruct.mutex.Lock()
	if !testStruct.stopped {
		t.Fatal("component not stopped")
	}
}

func TestInterruptByOsSignal(t *testing.T) {
	testStruct := &bootProcessesComponent{}
	ts := newTestSession(testStruct)

	go func() {
		time.Sleep(2 * time.Second)
		ts.changeMutex.Lock()
		defer ts.changeMutex.Unlock()
		ts.Session.option.shutdownChannel <- interruptSignal
	}()

	err := ts.Go()
	if err != nil {
		t.FailNow()
	}

	time.Sleep(1 * time.Second)
	defer testStruct.mutex.Unlock()
	testStruct.mutex.Lock()
	if !testStruct.stopped {
		t.Fatal("component not stopped")
	}
}

func TestResolveComponentError(t *testing.T) {
	testStruct := &bootMissingDependencyComponent{}
	ts := newTestSession(testStruct)

	err := ts.Go()
	if err == nil || err.Error() != "Error dependency field is not a pointer receiver <bootMissingDependencyComponent.WireFails>" {
		t.Fatal("resolve dependency error must result in an exit with proper error message")
	}
}

func TestRegister(t *testing.T) {
	type args struct {
		create func() Component
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Success",
			args: args{
				create: func() Component {
					return &bootTestComponent{}
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestSession()
			err := ts.register(DefaultName, tt.args.create, false)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

//nolint:dupl // duplication okay
func TestRegisterWithPanic(t *testing.T) {
	type args struct {
		name    string
		create  func() Component
		started bool
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "WithNoName",
			args: args{
				name: "",
				create: func() Component {
					return &bootTestComponent{}
				},
			},
			err: errSessionRegisterNameOrFunction,
		},
		{
			name: "WithoutFactoryFunction",
			args: args{
				name:   "Test",
				create: nil,
			},
			err: errSessionRegisterNameOrFunction,
		},
		{
			name: "BootAlreadyStarted",
			args: args{
				name: "Test",
				create: func() Component {
					return &bootProcessesComponent{}
				},
				started: true,
			},
			err: errSessionRegisterComponentOutsideInitialize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestSession(&bootProcessesComponent{})
			if tt.args.started {
				go func() {
					err := ts.Go()
					if err != nil {
						t.Error("Component test failed")
						return
					}
				}()
				time.Sleep(2 * time.Second)
			}
			err := ts.register(tt.args.name, tt.args.create, false)
			if !errors.Is(err, tt.err) {
				t.Error(err)
			}
		})
	}
}

func TestOverride(t *testing.T) {
	type args struct {
		create func() Component
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Success",
			args: args{
				create: func() Component {
					return &bootTestComponent{}
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestSession()
			err := ts.register(DefaultName, tt.args.create, true)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

//nolint:dupl // duplication okay
func TestOverrideWithPanic(t *testing.T) {
	type args struct {
		name    string
		create  func() Component
		started bool
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "WithNoName",
			args: args{
				name: "",
				create: func() Component {
					return &bootTestComponent{}
				},
			},
			err: errSessionRegisterNameOrFunction,
		},
		{
			name: "WithoutFactoryFunction",
			args: args{
				name:   "Test",
				create: nil,
			},
			err: errSessionRegisterNameOrFunction,
		},
		{
			name: "BootAlreadyStarted",
			args: args{
				name: "Test",
				create: func() Component {
					return &bootProcessesComponent{}
				},
				started: true,
			},
			err: errSessionRegisterComponentOutsideInitialize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestSession(&bootProcessesComponent{})
			if tt.args.started {
				go func() {
					err := ts.Go()
					if err != nil {
						t.Error("Component test failed")
						return
					}
				}()
				time.Sleep(2 * time.Second)
			}
			err := ts.register(tt.args.name, tt.args.create, true)
			if !errors.Is(err, tt.err) {
				t.Error(err)
			}
		})
	}
}

func TestPhaseString(t *testing.T) {
	tests := []struct {
		name string
		p    phase
		want string
	}{
		{name: "none", p: math.MaxUint8, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoot(t *testing.T) {
	globalSession = NewSession(UnitTestFlag)
	Register(func() Component {
		return &bootTestComponent{}
	})
	Override(func() Component {
		return &bootProcessesComponent{}
	})
	go func() {
		time.Sleep(time.Second)
		err := Shutdown()
		if err != nil {
			t.Fail()
		}
	}()
	err := Go()
	if err != nil {
		t.Fatal("boot failed")
	}
}

func TestBootFail(t *testing.T) {
	globalSession = NewSession(UnitTestFlag)
	Register(func() Component {
		return &bootMissingDependencyComponent{}
	})
	err := Go()
	if err == nil {
		t.Failed()
	}
}

func TestRegisterFail(t *testing.T) {
	globalSession = NewSession(UnitTestFlag)
	Register(func() Component {
		return &bootTestComponent{}
	})
	Override(func() Component {
		return &bootProcessesComponent{}
	})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := Shutdown()
				if err != nil {
					t.Fail()
				}
			}
		}()
		time.Sleep(time.Second)
		Register(func() Component {
			return &bootTestComponent{}
		})
		t.Failed()
	}()
	err := Go()
	if err != nil {
		t.Fatal("boot failed")
	}
}

func TestOverrideFail(t *testing.T) {
	globalSession = NewSession(UnitTestFlag)
	Register(func() Component {
		return &bootTestComponent{}
	})
	Override(func() Component {
		return &bootProcessesComponent{}
	})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := Shutdown()
				if err != nil {
					t.Fail()
				}
			}
		}()
		time.Sleep(time.Second)
		Override(func() Component {
			return &bootTestComponent{}
		})
		t.Failed()
	}()
	err := Go()
	if err != nil {
		t.Fatal("boot failed")
	}
}
