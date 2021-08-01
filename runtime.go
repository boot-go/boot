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

// runtime contains the configuration settings, which must be available globally for all components
// at the same time. Do not use this for component configurations.
type runtime struct {
	modes []Flag
}

// Flag describes a special behaviour of a component
type Flag string

// Runtime is a standard component, which is used to alternate the component behaviour at runtime.
type Runtime interface {
	HasFlag(flag Flag) bool
}

const (
	// StandardFlag is set when the component is started in unit test mode.
	StandardFlag Flag = "standard"
	// UnitTestFlag is set when the component is started in unit test mode.
	UnitTestFlag Flag = "unit test"
	// FunctionalTestFlag is set when the component is started in functional test mode.
	FunctionalTestFlag Flag = "functional test"
)

var _ Component = (*runtime)(nil) // Verify conformity to Component

func (r *runtime) Init() {}

func (r *runtime) HasFlag(mode Flag) bool {
	for _, m := range r.modes {
		if m == mode {
			return true
		}
	}
	return false
}
