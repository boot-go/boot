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
	"math"
	"os"
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
}

func (t *bootProcessesComponent) Init() error {
	t.block = make(chan bool, 1)
	t.stopped = false
	return nil
}

func (t *bootProcessesComponent) Start() error {
	<-t.block
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

type bootPhaseComponent struct {
	block                   chan bool
	onStart, onStop, onInit bool
	phase                   Phase
}

func (t *bootPhaseComponent) Init() error {
	t.block = make(chan bool, 1)
	if t.onInit {
		phase = t.phase
	}
	return nil
}

func (t *bootPhaseComponent) Start() error {
	if t.onStart {
		phase = t.phase
	}
	if t.onStop {
		Shutdown()
		<-t.block
	}
	return nil
}

func (t *bootPhaseComponent) Stop() error {
	if t.onStop {
		phase = t.phase
	}
	t.block <- true
	return nil
}

func TestBootGo(t *testing.T) {
	testStruct := &bootTestComponent{}
	setupTest()
	registerTestComponent(testStruct)

	err := Go()
	if err != nil {
		t.FailNow()
	}

	if !testStruct.initCalled ||
		!testStruct.startCalled ||
		testStruct.stopCalled {
		t.Fail()
	}
	tearDown()
}

func TestBootAlreadyRegisteredComponent(t *testing.T) {
	testStruct := &bootTestComponent{}
	setupTest()
	registerTestComponent(testStruct, testStruct)

	err := Go()
	if err != nil && err.Error() == "go aborted because component github.com/boot-go/boot/bootTestComponent already registered under the name 'default'" {
		tearDown()
		return
	}
	t.Fatal("error expected on already registered component")
}

func TestBootFactoryFail(t *testing.T) {
	err := Test(nil)
	if err == nil {
		t.FailNow()
	}
	tearDown()
}

func TestBootWithErrorComponent(t *testing.T) {

	tests := []struct {
		name    string
		content any
		err     error
	}{
		{name: "string content", content: "string content", err: errors.New("initializing default:github.com/boot-go/boot/bootPanicComponent failed with message: string content")},
		{name: "error content", content: errors.New("error content"), err: errors.New("initializing default:github.com/boot-go/boot/bootPanicComponent failed with error: error content")},
		{name: "other content", content: 0, err: errors.New("initializing default:github.com/boot-go/boot/bootPanicComponent failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testStruct := &bootPanicComponent{content: tt.content}
			setupTest()
			overrideTestComponent(testStruct)
			err := Go()
			if err == nil || err.Error() != tt.err.Error() {
				t.Errorf("Expected '%s' but found '%s'", tt.err.Error(), err.Error())
			}
		})
		tearDown()
	}
}

func TestBootShutdown(t *testing.T) {
	testStruct := &bootProcessesComponent{}
	setupTest()
	overrideTestComponent(testStruct)

	go func() {
		time.Sleep(5 * time.Second)
		Shutdown()
	}()

	err := Go()
	if err != nil {
		t.FailNow()
	}

	time.Sleep(2 * time.Second)
	if !testStruct.stopped {
		t.Fatal("Component not stopped")
	}

	tearDown()
}

func TestShutdownByOSSignal(t *testing.T) {
	testStruct := &bootProcessesComponent{}

	go func() {
		time.Sleep(2 * time.Second)
		shutdownChannel <- os.Kill
	}()

	err := Test(testStruct)
	if err != nil {
		t.FailNow()
	}

	time.Sleep(1 * time.Second)
	if !testStruct.stopped {
		t.Fatal("Component not stopped")
	}

	tearDown()
}

func TestResolveComponentError(t *testing.T) {
	testStruct := &bootMissingDependencyComponent{}
	err := Test(testStruct)
	if err == nil || err.Error() != "Error dependency field is not a pointer receiver <bootMissingDependencyComponent.WireFails>" {
		t.Fatal("resolve dependency error must result in an exit with proper error message")
	}
	tearDown()
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
			setupTest()
			Register(tt.args.create)
		})
		tearDown()
	}
}

func TestRegisterWithPanic(t *testing.T) {
	type args struct {
		name    string
		create  func() Component
		started bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "WithNoName",
			args: args{
				name: "",
				create: func() Component {
					return &bootTestComponent{}
				},
			},
		},
		{
			name: "WithoutFactoryFunction",
			args: args{
				name:   "Test",
				create: nil,
			},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("panic was not emitted")
				}
			}()
			if tt.args.started {
				go func() {
					err := Test(&bootProcessesComponent{})
					if err != nil {
						t.Error("Component test failed")
						return
					}
				}()
				time.Sleep(2 * time.Second)
			}
			RegisterName(tt.args.name, tt.args.create)
		})
		tearDown()
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
			setupTest()
			Override(tt.args.create)
		})
		tearDown()
	}
}

func TestOverrideWithPanic(t *testing.T) {
	type args struct {
		name    string
		create  func() Component
		started bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "WithNoName",
			args: args{
				name: "",
				create: func() Component {
					return &bootTestComponent{}
				},
			},
		},
		{
			name: "WithoutFactoryFunction",
			args: args{
				name:   "Test",
				create: nil,
			},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("panic was not emitted")
				}
			}()
			if tt.args.started {
				go func() {
					err := Test(&bootProcessesComponent{})
					if err != nil {
						t.Error("Component test failed")
						return
					}
				}()
				time.Sleep(2 * time.Second)
			}
			OverrideName(tt.args.name, tt.args.create)
		})
		tearDown()
	}
}

func TestPhaseWhenStartComponents(t *testing.T) {
	type args struct {
		entries []*entry
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "BootPhaseError",
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase = Initializing
			if err := startComponents(tt.args.entries); (err != nil) != tt.wantErr {
				t.Errorf("startComponents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		tearDown()
	}
}

func TestPhaseWhenStopComponents(t *testing.T) {
	type args struct {
		entries []*entry
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "BootPhaseError",
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase = Initializing
			if err := stopComponents(tt.args.entries); (err != nil) != tt.wantErr {
				t.Errorf("startComponents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		tearDown()
	}
}

func TestPhaseErrorWhenRun(t *testing.T) {
	type args struct {
		factoryList func() []factory
	}

	tests := []struct {
		name    string
		args    args
		want    []*entry
		phase   Phase
		wantErr string
	}{
		{
			name: "BootPhaseError",
			args: args{factoryList: func() []factory {
				return factories
			}},
			wantErr: "current boot phase stopping doesn't match expected boot phase initialization",
			phase:   Stopping,
		},
		{
			name: "BootPhaseAfterStartError",
			args: args{factoryList: func() []factory {
				return append(factories, factory{
					create: func() Component {
						return &bootPhaseComponent{phase: Initializing, onInit: true}
					},
					name:     "phase_hack_component",
					override: false,
				})
			}},
			wantErr: "current boot phase initialization doesn't match expected boot phase booting",
			phase:   Initializing,
		},
		{
			name: "BootPhaseAfterStopError",
			args: args{factoryList: func() []factory {
				return append(factories, factory{
					create: func() Component {
						return &bootPhaseComponent{phase: Exiting, onStop: true}
					},
					name:     "phase_hack_component",
					override: false,
				})
			}},
			wantErr: "current boot phase exiting doesn't match expected boot phase stopping",
			phase:   Initializing,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()
			phase = tt.phase
			_, err := run(tt.args.factoryList())
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
		tearDown()
	}
}

func TestPhaseString(t *testing.T) {
	tests := []struct {
		name string
		p    Phase
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
	tearDown()
}

func tearDown() {
	// cool down required to avoid race conditions while testing the most outer functions.
	time.Sleep(2 * time.Second)
}
