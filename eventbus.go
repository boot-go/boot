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
	"reflect"
	"sync"
)

// eventBus internal implementation
type eventBus struct {
	handlers map[string][]*eventHandler // contains all handler for a given type
	lock     sync.RWMutex               // a lock for the handler map
}

// EventBus provides the ability to decouple components. It is designed as a replacement for direct
// methode calls, so components can subscribe for messages produced by other components.
type EventBus interface {
	// Subscribe subscribes to a message type.
	// Returns error if handler fails.
	Subscribe(handler interface{}) error
	// Unsubscribe removes handler defined for a message type.
	// Returns error if there are no handlers subscribed to the message type.
	Unsubscribe(handler interface{}) error
	// Publish executes handler defined for a message type.
	Publish(event interface{}) (err error)
	// HasMessageHandler returns true if exists any handler subscribed to the message type.
	HasMessageHandler(event interface{}) bool
}

var _ Component = (*eventBus)(nil) // Verify conformity to Component

// Init is described in the Component interface
func (bus *eventBus) Init() {
}

// eventHandler contains the reference to one subscribed member
type eventHandler struct {
	handler       reflect.Value
	qualifiedName string
}

// newEventbus returns new an eventBus.
func newEventbus() *eventBus {
	return &eventBus{
		make(map[string][]*eventHandler),
		sync.RWMutex{},
	}
}

// Subscribe subscribes to a message type.
// Returns error if `fn` is not a function.
func (bus *eventBus) Subscribe(handler interface{}) error {
	if handler == nil {
		return fmt.Errorf("handler must not be nil")
	}
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		return fmt.Errorf("%s is not of type reflect.Func", reflect.TypeOf(handler).Kind())
	}
	p := reflect.ValueOf(handler)
	if p.Type().NumIn() != 1 {
		return errors.New("unsubscribe error while because number of arguments expected 1, but found " + fmt.Sprintf("%v", p.Type().NumIn()))
	}
	eventType, err := getEventType(p)
	if err != nil {
		return fmt.Errorf("%s seems not to be an regular message", eventType)
	}
	defer bus.lock.Unlock()
	bus.lock.Lock()
	bus.handlers[eventType] = append(bus.handlers[eventType], &eventHandler{
		handler:       reflect.ValueOf(handler),
		qualifiedName: QualifiedName(handler),
	})
	return nil
}

func getEventType(p reflect.Value) (string, error) {
	path := p.Type().In(0).PkgPath()
	name := p.Type().In(0).Name()
	if len(path) == 0 && len(name) == 0 {
		return "", errors.New("couldn't determiner the message type")
	}
	return path + "/" + name, nil
}

// HasMessageHandler returns true if exists any subscribed message handler.
func (bus *eventBus) HasMessageHandler(message interface{}) bool {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	eventType := QualifiedName(message)
	_, ok := bus.handlers[eventType]
	if ok {
		return len(bus.handlers[eventType]) > 0
	}
	return false
}

// Unsubscribe removes handler defined for a message type.
// Returns error if there are no handlers subscribed to the message type.
func (bus *eventBus) Unsubscribe(handler interface{}) error {
	if handler == nil {
		return fmt.Errorf("handler must not be nil")
	}
	// eventType := QualifiedName(message)
	p := reflect.ValueOf(handler)
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		return fmt.Errorf("%s is not of type reflect.Func", reflect.TypeOf(handler).Kind())
	}
	if p.Type().NumIn() != 1 {
		return errors.New("unsubscribe error because number of arguments expected 1, but found " + fmt.Sprintf("%v", p.Type().NumIn()))
	}
	eventType, err := getEventType(p)
	if err != nil {
		return fmt.Errorf("%s seems not to be an regular message", eventType)
	}
	bus.lock.Lock()
	defer bus.lock.Unlock()
	if _, ok := bus.handlers[eventType]; ok && len(bus.handlers[eventType]) > 0 {
		if ok := bus.removeHandler(eventType, bus.findHandler(eventType, handler)); !ok {
			return errors.New("handler not found to remove")
		}
		return nil
	}
	return fmt.Errorf("eventType %s doesn't exist", eventType)
}

// Publish executes handler defined for a message type. Any additional argument will be transferred to the handler.
func (bus *eventBus) Publish(event interface{}) (err error) {
	// if the bus is processing already, the upcoming messages will be queued
	var eventType string
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = fmt.Errorf("unsupported error type found %s", QualifiedName(v))
			}
		}
		if err != nil {
			Logger.Error.Printf("publishing event %s failed: Error: %s\n", eventType, err.Error())
		}
	}()
	if event == nil {
		return errors.New("event must not be nil")
	}
	eventType = QualifiedName(event)
	Logger.Debug.Printf("publishing event %s\n", eventType)
	if handlers, ok := bus.handlers[eventType]; ok && 0 < len(handlers) {
		// Handlers slice may be changed by removeHandler and Unsubscribe during iteration,
		// so make a copy and iterate the copied slice.
		bus.lock.RLock()
		copyHandlers := make([]*eventHandler, len(handlers))
		copy(copyHandlers, handlers)
		bus.lock.RUnlock()
		for _, handler := range copyHandlers {
			passedArguments := bus.prepare(event)
			handler.handler.Call(passedArguments)
		}
	}
	return err
}

func (bus *eventBus) removeHandler(eventType string, index int) bool {
	l := len(bus.handlers[eventType])

	if !(0 <= index && index < l) {
		return false
	}

	copy(bus.handlers[eventType][index:], bus.handlers[eventType][index+1:])
	bus.handlers[eventType][l-1] = nil // or the zero value of T
	bus.handlers[eventType] = bus.handlers[eventType][:l-1]
	return true
}

func (bus *eventBus) findHandler(eventType string, removingandler interface{}) int {
	if _, ok := bus.handlers[eventType]; ok {
		for index, handler := range bus.handlers[eventType] {
			if handler.qualifiedName == QualifiedName(removingandler) {
				return index
			}
		}
	}
	return -1
}

func (bus *eventBus) prepare(event interface{}) []reflect.Value {
	callArgs := make([]reflect.Value, 1)
	callArgs[0] = reflect.ValueOf(event)
	return callArgs
}

// testableEventBus box for handlers and callbacks.
type testableEventBus struct {
	eventBus
}

// NewTestableEventBus can be used for unit testing.
func NewTestableEventBus() EventBus {
	return &testableEventBus{}
}
