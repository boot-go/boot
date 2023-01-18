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
	"reflect"
	"testing"
)

//nolint:funlen // Testdata
func TestBootWithWire(t *testing.T) {
	// ts := newTestSession()

	t1 := &testStruct1{}
	t2 := &testStruct2{}
	t3 := &testStruct3{}
	t4 := &testStruct4{}
	t5 := &testStruct5{}
	t6 := &testStruct6{}
	t7 := &testStruct7{}
	t8 := &testStruct8{}
	t9 := &testStruct9{}
	t10 := &testStruct10{}
	t11 := &testStruct11{}
	t12 := &testStruct12{}
	t13 := &testStruct13{}
	t14 := &testStruct14{}
	t15 := &testStruct15{}
	t16 := &testStruct16{}
	controls := []Component{t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14, t15, t16}

	registry := newRegistry()
	for _, control := range controls {
		err := registry.addItem(DefaultName, false, control)
		if err != nil {
			Logger.Error.Printf("registry.addItem() failed: %v", err)
		}
	}
	err := registry.addItem("test", false, t1)
	if err != nil {
		Logger.Error.Printf("registry.addItem() failed: %v", err)
	}

	getEntry := func(c *Component) *componentManager {
		cmpName := QualifiedName(*c)
		return registry.items[cmpName][DefaultName]
	}

	tests := []struct {
		name       string
		controller Component
		expected   Component
		err        string
	}{
		{
			name:       "No injection",
			controller: t1,
			expected:   &testStruct1{},
		},
		{
			name:       "Single injection",
			controller: t2,
			expected: &testStruct2{
				F: t1,
			},
		},
		{
			name:       "Multiple injections",
			controller: t3,
			expected: &testStruct3{
				F: t1,
				G: t2,
			},
		},
		{
			name:       "Failed injection into unexported variable",
			controller: t4,
			err:        "Error dependency value cannot be set into <testStruct4.f>",
		},
		{
			name:       "Injection into interface",
			controller: t5,
			expected: &testStruct5{
				F: t1,
			},
		},
		{
			name:       "Failed injection not unique",
			controller: t6,
			err:        "Error multiple dependency values found for <default:testStruct6.F>",
		},
		{
			name:       "Failed injection for unrecognized component",
			controller: t7,
			err:        "Error dependency value not found for <default:testStruct7.F>",
		},
		{
			name:       "Failed injection non pointer receiver",
			controller: t8,
			err:        "Error dependency field is not a pointer receiver <testStruct8.F>",
		},
		{
			name:       "Single injection by name",
			controller: t9,
			expected: &testStruct9{
				F: t1,
			},
		},
		{
			name:       "Single injection by unknown name",
			controller: t10,
			err:        "Error dependency value not found for <unknown:testStruct10.F>",
		},
		{
			name:       "Single injection with unparsable name",
			controller: t11,
			err:        "Error field contains unparsable tag  <testStruct11.F `wire,name:`>",
		},
		{
			name:       "Traverse injection failed",
			controller: t12,
			err:        "Error field contains unparsable tag  <testStruct11.F `wire,name:`>",
		},
		{
			name:       "Tag format error",
			controller: t13,
			err:        "Error field contains unparsable tag  <testStruct13.F `wire,name:default:unsupported`>",
		},
		{
			name:       "Injection failed due tag format error",
			controller: t14,
			err:        "Error field contains unparsable tag  <testStruct13.F `wire,name:default:unsupported`>",
		},
		{
			name:       "Injection failed due init error",
			controller: t15,
			err:        "failed to initialize component default:github.com/boot-go/boot/testStruct15 - reason: fail-15",
		},
		{
			name:       "Injection failed due wired init error",
			controller: t16,
			err:        "failed to initialize component default:github.com/boot-go/boot/testStruct15 - reason: fail-15",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := resolveDependency(getEntry(&test.controller), registry)
			if test.err == "" {
				if err != nil {
					t.Fail()
				}
				if !reflect.DeepEqual(test.controller, test.expected) {
					t.Fail()
				}
			} else if err != nil && err.Error() != test.err {
				t.Errorf("error occurred\nexpected: %s\n     got: %v", test.err, err.Error())
			}
		})
	}
}

type testInterface1 interface {
	do1()
}

type testInterface2 interface {
	do2()
}

//nolint:unused // for testing purpose nolint:unused
type testStruct1 struct {
	a int
	B int
	c string
	d any
	e []any
}

func (t testStruct1) do1() {}

func (t testStruct1) Init() error { return nil }

func (t testStruct1) Start() error { return nil }

func (t testStruct1) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct2 struct {
	a int
	B int
	c string
	d any
	e []any
	F *testStruct1 `boot:"wire"`
}

func (t testStruct2) do2() {}

func (t testStruct2) Init() error { return nil }

func (t testStruct2) Start() error { return nil }

func (t testStruct2) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct3 struct {
	a int
	B int
	c string
	d any
	e []any        //nolint:unused // for testing purpose nolint:unused
	F *testStruct1 `boot:"wire"`
	G *testStruct2 `boot:"wire"`
}

func (t testStruct3) Init() error { return nil }

func (t testStruct3) Start() error { return nil }

func (t testStruct3) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct4 struct {
	a int //nolint:unused // for testing purpose nolint:unused
	B int
	c string
	d any
	e []any
	f *testStruct1 `boot:"wire"` //nolint:unused // for testing purpose nolint:unused
}

func (t testStruct4) Init() error { return nil }

func (t testStruct4) Start() error { return nil }

func (t testStruct4) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct5 struct {
	a int
	B int
	c string //nolint:unused // for testing purpose nolint:unused
	d any
	e []any
	F testInterface1 `boot:"wire"`
}

func (t testStruct5) Init() error { return nil }

func (t testStruct5) Start() error { return nil }

func (t testStruct5) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct6 struct {
	a int
	B int
	c string
	d any
	e []any
	F testInterface2 `boot:"wire"`
}

func (t testStruct6) do2() {}

func (t testStruct6) Init() error { return nil }

func (t testStruct6) Start() error { return nil }

func (t testStruct6) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct7 struct {
	a int
	B int
	c string
	d any
	e []any
	F *string `boot:"wire"`
}

func (t testStruct7) Init() error { return nil }

func (t testStruct7) Start() error { return nil }

func (t testStruct7) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct8 struct {
	a int
	B int
	c string
	d any
	e []any
	F testStruct1 `boot:"wire"`
}

func (t testStruct8) Init() error { return nil }

func (t testStruct8) Start() error { return nil }

func (t testStruct8) Stop() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct9 struct {
	a int
	B int
	c string
	d any
	e []any
	F *testStruct1 `boot:"wire,name:test"`
}

func (t testStruct9) Init() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct10 struct {
	a int
	B int
	c string
	d any
	e []any
	F *testStruct1 `boot:"wire,name:unknown"`
}

func (t testStruct10) Init() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct11 struct {
	a int //nolint:unused // for testing purpose nolint:unused
	B int
	c string
	d any
	e []any
	F *testStruct1 `boot:"wire,name:"`
}

func (t testStruct11) Init() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct12 struct {
	a int //nolint:unused // for testing purpose nolint:unused
	B int
	c string
	d any
	e []any
	F *testStruct11 `boot:"wire,name:default"`
}

func (t testStruct12) Init() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct13 struct {
	a int
	B int
	c string
	d any
	e []any
	F *testStruct11 `boot:"wire,name:default:unsupported"`
}

func (t testStruct13) Init() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct14 struct {
	a int
	B int
	c string
	d any
	e []any         //nolint:unused // for testing purpose nolint:unused
	F *testStruct13 `boot:"wire"`
}

func (t testStruct14) Init() error { return nil }

//nolint:unused // for testing purpose nolint:unused
type testStruct15 struct {
	a int
	B int
	c string
	d any
	e []any //nolint:unused // for testing purpose nolint:unused
}

func (t testStruct15) Init() error { return errors.New("fail-15") }

//nolint:unused // for testing purpose nolint:unused
type testStruct16 struct {
	a int
	B int
	c string
	d any
	e []any         //nolint:unused // for testing purpose nolint:unused
	F *testStruct15 `boot:"wire"`
}

func (t testStruct16) Init() error { return nil }
