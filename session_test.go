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
	"testing"
)

func TestSessionNextPhaseAfter(t *testing.T) {
	type fields struct {
		phase phase
	}
	type args struct {
		expected phase
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "next phase after initializing",
			fields: fields{
				phase: initializing,
			},
			args:    args{initializing},
			wantErr: false,
		},
		{
			name: "next phase after booting",
			fields: fields{
				phase: booting,
			},
			args:    args{booting},
			wantErr: false,
		},
		{
			name: "next phase after running",
			fields: fields{
				phase: running,
			},
			args:    args{running},
			wantErr: false,
		},
		{
			name: "next phase after stopping",
			fields: fields{
				phase: stopping,
			},
			args:    args{stopping},
			wantErr: false,
		},
		{
			name: "next phase after stopping",
			fields: fields{
				phase: exiting,
			},
			args:    args{exiting},
			wantErr: false,
		},
		{
			name: "fail next phase after stopping",
			fields: fields{
				phase: exiting,
			},
			args:    args{initializing},
			wantErr: true,
		},
		{
			name: "fail with unknown next phase after exiting",
			fields: fields{
				phase: 100,
			},
			args:    args{101},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSession(UnitTestFlag)
			s.phase = tt.fields.phase
			if err := s.nextPhaseAfter(tt.args.expected); (err != nil) != tt.wantErr {
				t.Errorf("nextPhaseAfter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionRegister(t *testing.T) { //nolint:dupl // duplication accepted
	type fields struct {
		phase phase
	}
	type args struct {
		name   string
		create func() Component
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "successful",
			fields: fields{
				phase: initializing,
			},
			args: args{
				name: DefaultName,
				create: func() Component {
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name: "error when no name or func ",
			fields: fields{
				phase: initializing,
			},
			args: args{
				name:   "",
				create: nil,
			},
			wantErr: errSessionRegisterNameOrFunction,
		},
		{
			name: "error when no name or func ",
			fields: fields{
				phase: initializing,
			},
			args: args{
				name:   "",
				create: nil,
			},
			wantErr: errSessionRegisterNameOrFunction,
		},
		{
			name: "error when phase is not initializing",
			fields: fields{
				phase: running,
			},
			args: args{
				name: DefaultName,
				create: func() Component {
					return nil
				},
			},
			wantErr: errSessionRegisterComponentOutsideInitialize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSession(UnitTestFlag)
			s.phase = tt.fields.phase
			if err := s.RegisterName(tt.args.name, tt.args.create); err != tt.wantErr { //nolint:errorlint // using errors.Is(..) will fail test
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionRegisterDefault(t *testing.T) {
	type fields struct {
		phase phase
	}
	type args struct {
		create func() Component
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "successful",
			fields: fields{
				phase: initializing,
			},
			args: args{
				create: func() Component {
					return nil
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSession(UnitTestFlag)
			s.phase = tt.fields.phase
			if err := s.Register(tt.args.create); err != tt.wantErr { //nolint:errorlint // errors.Is(...) will fail
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionOverride(t *testing.T) { //nolint:dupl // duplication accepted
	type fields struct {
		phase phase
	}
	type args struct {
		name   string
		create func() Component
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "successful",
			fields: fields{
				phase: initializing,
			},
			args: args{
				name: DefaultName,
				create: func() Component {
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name: "error when no name or func ",
			fields: fields{
				phase: initializing,
			},
			args: args{
				name:   "",
				create: nil,
			},
			wantErr: errSessionRegisterNameOrFunction,
		},
		{
			name: "error when no name or func ",
			fields: fields{
				phase: initializing,
			},
			args: args{
				name:   "",
				create: nil,
			},
			wantErr: errSessionRegisterNameOrFunction,
		},
		{
			name: "error when phase is not initializing",
			fields: fields{
				phase: running,
			},
			args: args{
				name: DefaultName,
				create: func() Component {
					return nil
				},
			},
			wantErr: errSessionRegisterComponentOutsideInitialize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSession(UnitTestFlag)
			s.phase = tt.fields.phase
			if err := s.OverrideName(tt.args.name, tt.args.create); err != tt.wantErr { //nolint:errorlint // errors.Is(...) will fail
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionOverrideDefault(t *testing.T) {
	type fields struct {
		phase phase
	}
	type args struct {
		create func() Component
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "successful",
			fields: fields{
				phase: initializing,
			},
			args: args{
				create: func() Component {
					return nil
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSession(UnitTestFlag)
			s.phase = tt.fields.phase
			if err := s.Override(tt.args.create); err != tt.wantErr { //nolint:errorlint // using errors.Is(..) will fail test
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type eventbusActivationTest struct {
	Eventbus               EventBus `boot:"wire"`
	initSubscribeReturnErr error
}

func (e *eventbusActivationTest) Init() error {
	_ = e.Eventbus.Subscribe(func(t testEvent) error {
		return e.initSubscribeReturnErr
	})
	return e.Eventbus.Publish(testEvent{})
}

type eventbusActivationProcessTest struct {
	Eventbus               EventBus `boot:"wire"`
	initSubscribeReturnErr error
}

func (e *eventbusActivationProcessTest) Init() error { return nil }

func (e *eventbusActivationProcessTest) Start() error {
	return e.Eventbus.Publish(testEvent{})
}

func (e *eventbusActivationProcessTest) Stop() error { return nil }

func TestSessionRunEventbusActivationFail(t *testing.T) {
	type args struct {
		create func() Component
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "successful",
			args: args{
				create: func() Component {
					return &eventbusActivationTest{
						initSubscribeReturnErr: nil,
					}
				},
			},
			wantErr: nil,
		},
		{
			name: "successful process component",
			args: args{
				create: func() Component {
					return &eventbusActivationProcessTest{
						initSubscribeReturnErr: nil,
					}
				},
			},
			wantErr: nil,
		},
		{
			name: "unsuccessful",
			args: args{
				create: func() Component {
					return &eventbusActivationTest{
						initSubscribeReturnErr: errors.New("fail"),
					}
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSession(UnitTestFlag)
			err := s.Register(tt.args.create)
			if err != nil {
				t.Fail()
			}
			err = s.Go()
			if err != tt.wantErr { //nolint:errorlint // using errors.Is(..) will fail test
				t.Errorf("Go() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testPhase struct {
	s     *testSession
	init  phase
	start phase
	stop  phase
}

func (t *testPhase) Init() error {
	t.s.phase = t.init
	return nil
}

func (t *testPhase) Start() error {
	t.s.phase = t.start
	return nil
}

func (t *testPhase) Stop() error {
	t.s.phase = t.stop
	return nil
}

func TestSessionRun(t *testing.T) {
	tests := []struct {
		name         string
		initPhase    phase
		bootPhase    phase
		runningPhase phase
		wantErr      string
	}{
		{
			name:      "fail boot",
			initPhase: exiting,
			wantErr:   "current boot phase exiting doesn't match expected boot phase initialization",
		},
		{
			name:      "fail init",
			initPhase: initializing,
			bootPhase: exiting,
			wantErr:   "current boot phase exiting doesn't match expected boot phase booting",
		},
		{
			name:         "fail start",
			initPhase:    initializing,
			bootPhase:    booting,
			runningPhase: exiting,
			wantErr:      "current boot phase exiting doesn't match expected boot phase stopping",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSession()
			_ = s.registerTestComponent(&testPhase{
				s:     s,
				init:  tt.bootPhase,
				start: tt.runningPhase,
			})
			s.phase = tt.initPhase
			err := s.Go()
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("Go() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestSessionFailOnCreateNilComponents(t *testing.T) {
	t.Run("nil component", func(t *testing.T) {
		s := newTestSession(nil)
		_, err := s.createComponents()
		if err == nil {
			t.Errorf("createComponents() must throw an error")
			return
		}
	})
}

func TestSessionWithOptionsFailOnDoShutdown(t *testing.T) {
	t.Run("doShutdown fails", func(t *testing.T) {
		s := newTestSessionWithOptions(Options{
			DoShutdown: func() error {
				return errors.New("fail")
			},
		})
		err := s.Shutdown()
		if err == nil || err.Error() != "fail" {
			t.Fail()
		}
	})
}

func TestSessionWithOptionsFailOnDoMain(t *testing.T) {
	t.Run("doMain fails", func(t *testing.T) {
		s := newTestSessionWithOptions(Options{
			DoMain: func() error {
				return errors.New("fail")
			},
		})
		err := s.Go()
		if err == nil || err.Error() != "fail" {
			t.Fail()
		}
	})
}
