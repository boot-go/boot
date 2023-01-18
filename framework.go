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
	gort "runtime"
	"time"
)

// Component represent functional building blocks, which solve one specific purpose.
// They should be fail tolerant, recoverable, agnostic and decent.
//
// fail tolerant: Don't stop processing on errors.
// Example: A http request can still be processed, even when the logging server is not available anymore.
//
// recoverable: Try to recover from errors.
// E.g. a database component should try to reconnect after lost connection.
//
// agnostic: Behave the same in any environment.
// E.g. a key-value store component should work on a local development Session the same way as in a containerized environment.
//
// decent: Don't overload the developer with complexity.
// E.g. keep the interface and events as simple as possible. Less is often more.
type Component interface {
	// Init initializes data, set the default configuration, subscribe to events
	// or performs other kind of configuration.
	Init() error
}

// Process is a Component which has a processing functionality. This can be anything like a server,
// cron job or long-running process.
type Process interface {
	Component
	// Start is called as soon as all boot.Component components are initialized. The call should be
	// blocking until all processing is completed.
	Start() error
	// Stop is called to abort the processing and clean up resources. Pay attention that the
	// processing may already be stopped.
	Stop() error
}

const (
	// DefaultName is used when registering components without an explicit name.
	DefaultName = "default"
)

// globalSession is the one and only global variable
var globalSession *Session

func init() {
	globalSession = NewSession(StandardFlag)
}

// Register a default factory function.
func Register(create func() Component) {
	RegisterName(DefaultName, create)
}

// RegisterName registers a factory function with the given name.
func RegisterName(name string, create func() Component) {
	err := globalSession.RegisterName(name, create)
	if err != nil {
		panic(err)
	}
}

// Override a default factory function.
func Override(create func() Component) {
	OverrideName(DefaultName, create)
}

// OverrideName overrides a factory function with the given name.
func OverrideName(name string, create func() Component) {
	err := globalSession.OverrideName(name, create)
	if err != nil {
		panic(err)
	}
}

// Go the boot component framework. This starts the execution process.
func Go() error {
	startTime := time.Now()
	s := new(gort.MemStats)
	gort.ReadMemStats(s)
	// output some basic info
	const kilobyte = 1024
	const megabyte = kilobyte * 2
	Logger.Info.Printf("booting `boot-go %s` /// %s OS/%s ARCH/%s CPU/%d MEM/%dMB SYS/%dMB\n", version, gort.Version(), gort.GOOS, gort.GOARCH, gort.NumCPU(), s.Alloc/megabyte, s.Sys/megabyte)
	err := globalSession.Go()
	if err == nil {
		Logger.Info.Printf("shutdowned after %s\n", time.Since(startTime).String())
	} else {
		Logger.Error.Printf("shutdowned after %s with: %s\n", time.Since(startTime).String(), err.Error())
	}
	return err
}

// Shutdown boot-go componentManager. All components will be stopped. This is equivalent with
// issuing a SIGTERM on process level.
func Shutdown() {
	globalSession.Shutdown()
}
