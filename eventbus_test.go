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
	"math/rand"
	"testing"
	"time"
)

type testEvent struct {
}

func TestEventbusHasCallback(t *testing.T) {
	testcases := []struct {
		name    string
		topic   interface{}
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
			hasCallback := bus.HasMessageHandler(tc.topic)
			if hasCallback != !tc.wantErr {
				t.Errorf("eventBus.subscribe() , wantErr %v", tc.wantErr)
			}
		})
	}
}

func TestEventbusSubscribe(t *testing.T) {
	testcases := []struct {
		name    string
		topic   interface{}
		wantErr bool
	}{
		{
			name:    "success",
			topic:   func(event testEvent) {},
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
			bus := newEventbus()
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
		event   interface{}
		topic   interface{}
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
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newEventbus()
			err := bus.Start()
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

func TestEventbusPublishMultipleEvents(t *testing.T) {
	const events = 100
	testcases := []struct {
		name    string
		topic   interface{}
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
			err := bus.Start()
			if err != nil {
				t.Errorf("failed to start event bus: %v", err)
			}
			counter := 0
			err = bus.Subscribe(func(event testEvent) {
				time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
				counter++
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

			hasCallback := bus.HasMessageHandler(testEvent{})
			if !hasCallback != tc.wantErr {
				t.Errorf("eventBus.subscribe() , wantErr %v", tc.wantErr)
			}
			time.Sleep(time.Second)
			if counter != events {
				t.Fail()
			}
		})
	}
}

func TestEventbusUnsubscribe(t *testing.T) {
	f := func(event testEvent) {}
	testcases := []struct {
		name         string
		event        interface{}
		eventHandler interface{}
		wantErr      bool
	}{
		{
			name:         "success",
			event:        f,
			eventHandler: f,
			wantErr:      false,
		},
		{
			name:         "non existing subscription",
			event:        f,
			eventHandler: func(event testEvent) {},
			wantErr:      true,
		},
		{
			name:         "non Existing Subscription",
			event:        f,
			eventHandler: func(event int) {},
			wantErr:      true,
		},
		{
			name:         "wrong number arguments",
			event:        f,
			eventHandler: func(event1 int, event2 int) {},
			wantErr:      true,
		},
		{
			name:         "non Function",
			event:        f,
			eventHandler: "",
			wantErr:      true,
		},
		{
			name:         "no Arg Function",
			event:        f,
			eventHandler: func() {},
			wantErr:      true,
		},
		{
			name:         "irregular handler",
			event:        f,
			eventHandler: func(event []string) {},
			wantErr:      true,
		},
		{
			name:         "nil error",
			event:        f,
			eventHandler: nil,
			wantErr:      true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bus := newEventbus()
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
		tb := NewTestableEventBus()
		if tb == nil {
			t.FailNow()
		}
	})
}
