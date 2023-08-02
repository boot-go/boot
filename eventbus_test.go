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
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

type testEvent struct {
}

func TestEventbusHasCallback(t *testing.T) {
	testcases := []struct {
		name    string
		topic   any
		wantErr bool
	}{
		{
			name:    "success",
			topic:   testEvent{},
			wantErr: false,
		}, {
			name:    "callback does not exist",
			topic:   "test",
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newEventbus()
			if tc.topic != nil {
				err := bus.Subscribe(func(event testEvent) {})
				if err != nil {
					t.Errorf("eventBus.Subscribe() failed with %v", err)
				}
			}
			hasCallback := bus.HasHandler(tc.topic)
			if hasCallback != !tc.wantErr {
				t.Errorf("eventBus.subscribe() , wantErr %v", tc.wantErr)
			}
		})
	}
}

func TestEventbusSubscribe(t *testing.T) {
	testcases := []struct {
		name    string
		topic   any
		wantErr bool
	}{
		{
			name:    "success",
			topic:   func(event testEvent) {},
			wantErr: false,
		}, {
			name:    "success with return",
			topic:   func(event testEvent) error { return nil },
			wantErr: false,
		}, {
			name:    "wrong topic type string",
			topic:   "test",
			wantErr: true,
		}, {
			name:    "wrong topic type int",
			topic:   "test",
			wantErr: true,
		},
		{
			name:    "irregular handler",
			topic:   func(event []string) {},
			wantErr: true,
		},
		{
			name:    "irregular handler with return",
			topic:   func(event []string) string { return "" },
			wantErr: true,
		},
		{
			name:    "irregular handler with multiple return values",
			topic:   func(event []string) (error, string) { return nil, "" },
			wantErr: true,
		},
		{
			name:    "multiple param event",
			topic:   func(event1 testEvent, event2 testEvent) {},
			wantErr: true,
		},
		{
			name:    "nil error",
			topic:   nil,
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newTestableEventBus()
			err := bus.Subscribe(tc.topic)
			if (err != nil) != tc.wantErr {
				t.Errorf("eventBus.subscribe() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestEventbusPublish(t *testing.T) {
	testcases := []struct {
		name    string
		event   any
		topic   any
		wantErr bool
	}{
		{
			name:    "success",
			event:   testEvent{},
			topic:   func(event testEvent) {},
			wantErr: false,
		},
		{
			name:  "panic with string",
			event: testEvent{},
			topic: func(event testEvent) {
				panic("Test panic")
			},
			wantErr: true,
		},
		{
			name:  "panic with error",
			event: testEvent{},
			topic: func(event testEvent) {
				panic(errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name:  "panic with int",
			event: testEvent{},
			topic: func(event testEvent) {
				panic(0)
			},
			wantErr: true,
		},
		{
			name:    "missing event",
			event:   nil,
			topic:   func(event testEvent) {},
			wantErr: true,
		},
		{
			name:    "successfully event processing with error return",
			event:   testEvent{},
			topic:   func(event testEvent) error { return nil },
			wantErr: false,
		},
		{
			name:    "event processing failed",
			event:   nil,
			topic:   func(event testEvent) error { return errors.New("test fail") },
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newEventbus()
			err := bus.activate()
			if err != nil {
				t.Errorf("failed to start event bus: %v", err)
			}
			err = bus.Subscribe(tc.topic)
			if err != nil {
				t.Errorf("eventBus.subscribe() error = %v", err)
			}
			err = bus.Publish(tc.event)
			if (err != nil) != tc.wantErr {
				t.Errorf("eventBus.publish() , wantErr %v", tc.wantErr)
			}
		})
	}
}

func TestEventbusNotActivatedPublish(t *testing.T) {
	eventTriggered := false
	testcases := []struct {
		name            string
		event           any
		topic           any
		wantPublishErr  bool
		wantActivateErr bool
	}{
		{
			name:  "success",
			event: testEvent{},
			topic: func(event testEvent) {
				eventTriggered = true
			},
			wantPublishErr:  false,
			wantActivateErr: false,
		},
		{
			name:  "error",
			event: testEvent{},
			topic: func(event testEvent) error {
				eventTriggered = true
				return errors.New("fail")
			},
			wantPublishErr:  false,
			wantActivateErr: true,
		},
		{
			name:  "panic",
			event: testEvent{},
			topic: func(event testEvent) error {
				eventTriggered = true
				panic("test")
			},
			wantPublishErr:  false,
			wantActivateErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			eventTriggered = false
			bus := newEventbus()
			err := bus.Subscribe(tc.topic)
			if err != nil {
				t.Errorf("eventBus.subscribe() error = %v", err)
			}
			err = bus.Publish(tc.event)
			if (err != nil) != tc.wantPublishErr {
				t.Errorf("eventBus.publish() , wantErr %v", tc.wantPublishErr)
			}
			err = bus.activate()
			if (err != nil) != tc.wantActivateErr {
				t.Errorf("failed to start event bus: %v", err)
			}
			if !eventTriggered {
				t.Fatal("event not published after activation")
			}
		})
	}
}

func TestEventbusPublishError(t *testing.T) {
	testcases := []struct {
		name    string
		event   any
		topics  []any
		wantErr string
	}{
		{
			name:    "one topic fail",
			event:   testEvent{},
			topics:  []any{func(event testEvent) error { return errors.New("fail1") }},
			wantErr: "publish failed for event: github.com/boot-go/boot/testEvent [github.com/boot-go/boot.TestEventbusPublishError.func1: fail1]",
		},
		{
			name:    "two topics fail",
			event:   testEvent{},
			topics:  []any{func(event testEvent) error { return errors.New("fail1") }, func(event testEvent) error { return errors.New("fail2") }},
			wantErr: "publish failed for event: github.com/boot-go/boot/testEvent [github.com/boot-go/boot.TestEventbusPublishError.func2: fail1] [github.com/boot-go/boot.TestEventbusPublishError.func3: fail2]",
		},
		{
			name:    "two topics fail, one succeed",
			event:   testEvent{},
			topics:  []any{func(event testEvent) error { return errors.New("fail1") }, func(event testEvent) error { return nil }, func(event testEvent) error { return errors.New("fail2") }},
			wantErr: "publish failed for event: github.com/boot-go/boot/testEvent [github.com/boot-go/boot.TestEventbusPublishError.func4: fail1] [github.com/boot-go/boot.TestEventbusPublishError.func6: fail2]",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newEventbus()
			err := bus.activate()
			if err != nil {
				t.Errorf("failed to start event bus: %v", err)
			}
			for _, topic := range tc.topics {
				err = bus.Subscribe(topic)
				if err != nil {
					t.Errorf("eventBus.subscribe() error = %v", err)
				}
			}
			err = bus.Publish(tc.event)
			pErr, ok := err.(*PublishError) //nolint:errorlint // casting required
			if !ok {
				t.Errorf("expected PublishError() , but found %v", err)
			}
			if pErr.Error() != tc.wantErr {
				t.Errorf("eventBus.publish()\nwant: %v\n got: %v", tc.wantErr, pErr)
			}
		})
	}
}

func TestEventbusPublishMultipleEvents(t *testing.T) {
	const events = 100
	testcases := []struct {
		name    string
		topic   any
		wantErr bool
	}{
		{
			name:    "success",
			wantErr: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newEventbus()
			err := bus.activate()
			if err != nil {
				t.Errorf("failed to start event bus: %v", err)
			}
			var counter int32 = 0
			err = bus.Subscribe(func(event testEvent) {
				time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond) //nolint:gosec // sec for this test is irrelevant
				atomic.AddInt32(&counter, 1)
			})
			if (err != nil) != tc.wantErr {
				t.Errorf("eventBus.subscribe() error = %v, wantErr %v", err, tc.wantErr)
			}

			for i := 0; i < events; i++ {
				go func() {
					err := bus.Publish(testEvent{})
					if err != nil {
						t.Errorf("bus.Publish() failed with %v", err)
					}
				}()
			}

			hasCallback := bus.HasHandler(testEvent{})
			if !hasCallback != tc.wantErr {
				t.Errorf("eventBus.subscribe() , wantErr %v", tc.wantErr)
			}
			time.Sleep(time.Second)
			if atomic.LoadInt32(&counter) != events {
				t.Fail()
			}
		})
	}
}

func TestEventbusUnsubscribe(t *testing.T) {
	testEventFunction := func(event testEvent) {}
	testcases := []struct {
		name         string
		event        any
		eventHandler any
		wantErr      bool
	}{
		{
			name:         "success",
			event:        testEventFunction,
			eventHandler: testEventFunction,
			wantErr:      false,
		},
		{
			name:         "non existing subscription",
			event:        testEventFunction,
			eventHandler: func(event testEvent) {},
			wantErr:      true,
		},
		{
			name:         "non Existing Subscription",
			event:        testEventFunction,
			eventHandler: func(event int) {},
			wantErr:      true,
		},
		{
			name:         "wrong number arguments",
			event:        testEventFunction,
			eventHandler: func(event1 int, event2 int) {},
			wantErr:      true,
		},
		{
			name:         "non Function",
			event:        testEventFunction,
			eventHandler: "",
			wantErr:      true,
		},
		{
			name:         "no Arg Function",
			event:        testEventFunction,
			eventHandler: func() {},
			wantErr:      true,
		},
		{
			name:         "irregular handler",
			event:        testEventFunction,
			eventHandler: func(event []string) {},
			wantErr:      true,
		},
		{
			name:         "nil error",
			event:        testEventFunction,
			eventHandler: nil,
			wantErr:      true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newTestableEventBus()
			err := bus.Subscribe(tc.event)
			if err != nil {
				t.Errorf("bus.Subscribe() failed: %v", err)
			}
			err = bus.Unsubscribe(tc.eventHandler)
			if (err != nil) != tc.wantErr {
				t.Fail()
			}
		})
	}
}

func TestNewTestableEventBus(t *testing.T) {
	t.Run("new testable event bus", func(t *testing.T) {
		bus := newTestableEventBus()
		if bus == nil {
			t.FailNow()
		}
	})
}
