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
	"reflect"
	"sync"
)

// eventBus internal implementation
type eventBus struct {
	Runtime   Runtime                   `boot:"wire"`
	handlers  map[string][]*busListener // contains all handler for a given type
	lock      sync.RWMutex              // a lock for the handler map
	isStarted bool
	queue     []any // the queue which will receive the events until the init phase is  changed
}

// Handler is a function which has one argument. This argument is usually a published event. An error
// may be optional provided. E.g. func(e MyEvent) err
type Handler any

// Event is published and can be any type
type Event any

// EventBus provides the ability to decouple components. It is designed as a replacement for direct
// methode calls, so components can subscribe for messages produced by other components.
type EventBus interface {
	// Subscribe subscribes to a message type.
	// Returns error if handler fails.
	Subscribe(handler Handler) error
	// Unsubscribe removes handler defined for a message type.
	// Returns error if there are no handlers subscribed to the message type.
	Unsubscribe(handler Handler) error
	// Publish executes handler defined for a message type.
	Publish(event Event) (err error)
	// HasHandler returns true if exists any handler subscribed to the message type.
	HasHandler(event Handler) bool
}

var _ Component = (*eventBus)(nil) // Verify conformity to Component

// errors
var (
	ErrHandlerMustNotBeNil = errors.New("handler must not be nil")
	ErrEventMustNotBeNil   = errors.New("event must not be nil")
	ErrUnknownEventType    = errors.New("couldn't determiner the message type")
	ErrHandlerNotFound     = errors.New("handler not found to remove")
	ErrHandlerNotSupported = errors.New("handler function not supported")
	ErrPublishEventFailed  = errors.New("publish event failed")
)

// Init is described in the Component interface
func (bus *eventBus) Init() error {
	bus.isStarted = false
	return nil
}

func (bus *eventBus) activate() error {
	bus.isStarted = true
	// republishing queued events
	Logger.Debug.Printf("eventbus started with %d queued events\n", len(bus.queue))
	pubErr := newPublicError()
	for _, event := range bus.queue {
		err := bus.Publish(event)
		if err != nil {
			Logger.Error.Printf("publishing queued event failed on eventbus start: %v", err.Error())
			if p, ok := err.(*PublishError); ok { //nolint:errorlint // casting required
				pubErr.addPublishError(p)
			} else {
				Logger.Error.Printf("unrecoverable error occurred while activating eventbus %v", err)
				return err
			}
		}
	}
	if pubErr.hasErrors() {
		return pubErr
	}
	return nil
}

// busListener contains the reference to one subscribed member
type busListener struct {
	handler       reflect.Value
	qualifiedName string
	eventTypeName string
}

// newBusListener will validate the handler and return the name of the event type, which is provided
// as an argument to the handler
func newBusListener(handler Handler) (*busListener, error) {
	if handler == nil {
		return nil, ErrHandlerMustNotBeNil
	}
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		return nil, fmt.Errorf("handler is not a function - detail: %s is not of type reflect.Func", reflect.TypeOf(handler).Kind())
	}
	// validate argument
	argValue := reflect.ValueOf(handler)
	if argValue.Type().NumIn() != 1 {
		// return fmt.Errorf("%w: unsubscribe error while because number of arguments expected 1, but found ", ErrHandlerNotSupported)
		return nil, fmt.Errorf("handler function has unsupported argument, found %d but requires 1" + fmt.Sprintf("%v", argValue.Type().NumIn()))
	}
	// validate return value type
	switch argValue.Type().NumOut() {
	case 0:
	case 1:
		retType := argValue.Type().Out(0)
		if _, ok := reflect.New(retType).Interface().(*error); !ok {
			return nil, fmt.Errorf("handler function return type is not an error")
		}
	default:
		return nil, fmt.Errorf("handler function has more than one return value, found %d but requires 1" + fmt.Sprintf("%v", argValue.Type().NumOut()))
	}
	path := argValue.Type().In(0).PkgPath()
	name := argValue.Type().In(0).Name()
	if len(path) == 0 && len(name) == 0 {
		return nil, ErrUnknownEventType
	}
	eventTypeName := path + "/" + name

	return &busListener{
		handler:       argValue,
		qualifiedName: QualifiedName(handler),
		eventTypeName: eventTypeName,
	}, nil
}

// newEventbus returns new an eventBus.
func newEventbus() *eventBus {
	return &eventBus{
		handlers: make(map[string][]*busListener),
		lock:     sync.RWMutex{},
		queue:    nil,
	}
}

// Subscribe subscribes to a message type.
// Returns error if `fn` is not a function.
func (bus *eventBus) Subscribe(handler Handler) error {
	eventHandler, err := newBusListener(handler)
	if err != nil {
		errRet := fmt.Errorf("%s seems not to be a regular handler function: %w", QualifiedName(handler), err)
		Logger.Error.Printf(errRet.Error())
		return errRet
	}
	defer bus.lock.Unlock()
	bus.lock.Lock()
	bus.handlers[eventHandler.eventTypeName] = append(bus.handlers[eventHandler.eventTypeName], eventHandler)
	Logger.Debug.Printf("handler %s subscribed\n", eventHandler.qualifiedName)
	return nil
}

// HasHandler returns true if exists any subscribed message handler.
func (bus *eventBus) HasHandler(handler Handler) bool {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	eventType := QualifiedName(handler)
	_, ok := bus.handlers[eventType]
	if ok {
		return len(bus.handlers[eventType]) > 0
	}
	return false
}

// Unsubscribe removes handler defined for a message type.
// Returns error if there are no handlers subscribed to the message type.
func (bus *eventBus) Unsubscribe(handler Handler) error {
	eventHandler, err := newBusListener(handler)
	if err != nil {
		errRet := fmt.Errorf("%s seems not to be an regular handler function: %w", QualifiedName(handler), err)
		Logger.Error.Printf(errRet.Error())
		return errRet
	}
	bus.lock.Lock()
	defer bus.lock.Unlock()
	if _, ok := bus.handlers[eventHandler.eventTypeName]; ok && len(bus.handlers[eventHandler.eventTypeName]) > 0 {
		if ok := bus.removeHandler(eventHandler.eventTypeName, bus.findHandler(eventHandler.eventTypeName, handler)); !ok {
			return ErrHandlerNotFound
		}
		Logger.Debug.Printf("handler %s unsubscribed \n", eventHandler.qualifiedName)
		return nil
	}
	return fmt.Errorf("eventType %s doesn't exist", QualifiedName(handler))
}

// PublishError will be provided by the
type PublishError struct {
	failedListeners map[Event]map[*busListener]error
}

func newPublicError() *PublishError {
	return &PublishError{
		failedListeners: make(map[Event]map[*busListener]error),
	}
}

func (err *PublishError) hasErrors() bool {
	return len(err.failedListeners) > 0
}

func (err *PublishError) addPublishError(pubErr *PublishError) {
	for event, m := range pubErr.failedListeners {
		for key, pErr := range m {
			err.addError(event, key, pErr)
		}
	}
}

func (err *PublishError) addError(e Event, bl *busListener, pubErr error) {
	m := err.failedListeners[e]
	if m == nil {
		m = make(map[*busListener]error)
		err.failedListeners[e] = m
	}
	m[bl] = pubErr
}

// Error is used to confirm to the error interface
func (err *PublishError) Error() string {
	str := "publish failed for "
	for event, m := range err.failedListeners {
		str += fmt.Sprintf("event: %s", QualifiedName(event))
		for key, pErr := range m {
			str = fmt.Sprintf("%s [%s: %s]", str, key.qualifiedName, pErr.Error())
		}
	}

	return str
}

var _ error = (*PublishError)(nil) // force error to confirm to error interface

// Publish executes handler defined for a message type. Any additional argument will be transferred to the handler.
func (bus *eventBus) Publish(event Event) (err error) {
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
			Logger.Error.Printf("publishing event %s failed: %s\n", eventType, err.Error())
		}
	}()
	if event == nil {
		return ErrEventMustNotBeNil
	}
	eventType = QualifiedName(event)
	if !bus.isStarted {
		Logger.Debug.Printf("queuing event %s\n", eventType)
		bus.queue = append(bus.queue, event)
		return
	}
	Logger.Debug.Printf("publishing event %s\n", eventType)
	if handlers, ok := bus.handlers[eventType]; ok && 0 < len(handlers) {
		// Handlers slice may be changed by removeHandler and Unsubscribe during iteration,
		// so make a copy and iterate the copied slice.
		bus.lock.RLock()
		copyHandlers := make([]*busListener, len(handlers))
		copy(copyHandlers, handlers)
		bus.lock.RUnlock()
		pErr := bus.publish(event, copyHandlers)
		if pErr != nil {
			return pErr
		}
	}
	return err
}

// publish the event to all provided bus listeners
func (bus *eventBus) publish(event Event, listeners []*busListener) *PublishError {
	errPublish := newPublicError()
	for _, listener := range listeners {
		passedArguments := bus.prepare(event)
		ret := listener.handler.Call(passedArguments)
		// a handler may return an error... validation will verify
		if len(ret) == 1 {
			err, ok := ret[0].Interface().(error)
			if ok && err != nil {
				errPublish.addError(event, listener, err)
			}
		}
	}
	if len(errPublish.failedListeners) > 0 {
		return errPublish
	}
	return nil
}

func (bus *eventBus) removeHandler(eventTypeName string, index int) bool {
	l := len(bus.handlers[eventTypeName])

	if !(0 <= index && index < l) {
		return false
	}

	copy(bus.handlers[eventTypeName][index:], bus.handlers[eventTypeName][index+1:])
	bus.handlers[eventTypeName][l-1] = nil // or the zero value of T
	bus.handlers[eventTypeName] = bus.handlers[eventTypeName][:l-1]
	return true
}

func (bus *eventBus) findHandler(eventTypeName string, handler any) int {
	if _, ok := bus.handlers[eventTypeName]; ok {
		for index, h := range bus.handlers[eventTypeName] {
			if h.qualifiedName == QualifiedName(handler) {
				return index
			}
		}
	}
	return -1
}

func (bus *eventBus) prepare(event any) []reflect.Value {
	callArgs := make([]reflect.Value, 1)
	callArgs[0] = reflect.ValueOf(event)
	return callArgs
}

// testableEventBus box for handlers and callbacks.
type testableEventBus struct {
	eventBus
}

// NewTestableEventBus can be used for unit testing.
func NewTestableEventBus() *testableEventBus {
	return &testableEventBus{eventBus{
		Runtime: &runtime{
			modes: []Flag{UnitTestFlag},
		},
		handlers: make(map[string][]*busListener),
		lock:     sync.RWMutex{},
		queue:    nil,
	}}
}
