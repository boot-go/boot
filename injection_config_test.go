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
	"os"
	"reflect"
	"testing"
)

func TestBootWithWireConfig(t *testing.T) {
	t1 := &envTestStruct1{}
	t2 := &envTestStruct2{}
	t3 := &envTestStruct3{}
	t4 := &envTestStruct4{}
	t5 := &envTestStruct5{}
	t6 := &envTestStruct6{}
	t7 := &envTestStruct7{}
	t8 := &envTestStruct8{}
	t9 := &envTestStruct9{}
	t10 := &envTestStruct10{}
	t11 := &envTestStruct11{}
	t12 := &envTestStruct12{}
	t13 := &envTestStruct13{}
	t14 := &envTestStruct14{}
	controls := []Component{t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14}

	registry := newRegistry()
	for _, control := range controls {
		registry.addEntry(DefaultName, false, control)
	}

	getEntry := func(c *Component) *entry {
		cmpName := QualifiedName(*c)
		return registry.entries[cmpName][DefaultName]
	}

	tests := []struct {
		name       string
		controller Component
		setup      func()
		expected   Component
		err        string
	}{
		{
			name:       "simple configuration",
			controller: t1,
			setup: func() {
				os.Setenv("t1", "v1")
			},
			expected: &envTestStruct1{
				C: "v1",
			},
		},
		{
			name:       "missing environment variable",
			controller: t2,
			expected: &envTestStruct2{
				C: "",
			},
		},
		{
			name:       "missing environment variable will panic",
			controller: t3,
			err:        "Error failed to load configuration value for t3 <envTestStruct3.C>",
		},
		{
			name:       "misconfigured tag",
			controller: t4,
			expected: &envTestStruct4{
				C: "",
			},
		},
		{
			name:       "misconfigured tag name",
			controller: t5,
			err:        "Error dependency field has unsupported tag  <envTestStruct5.C `wi-re,key:t3,panic`>",
		},
		{
			name:       "missing tag value",
			controller: t6,
			err:        "Error unsupported configuration options found <envTestStruct6.C>",
		},
		{
			name:       "simple int configuration",
			controller: t7,
			setup: func() {
				os.Setenv("t7", "100")
			},
			expected: &envTestStruct7{
				B: 100,
			},
		},
		{
			name:       "wrong int configuration",
			controller: t8,
			setup: func() {
				os.Setenv("t8", "XYZ")
			},
			err: "Error failed to load configuration value for t8 <envTestStruct8.B>",
		},
		{
			name:       "simple bool configuration",
			controller: t9,
			setup: func() {
				os.Setenv("t9", "true")
			},
			expected: &envTestStruct9{
				F: true,
			},
		},
		{
			name:       "wrong bool configuration",
			controller: t10,
			setup: func() {
				os.Setenv("t10", "xyz")
			},
			err: "Error failed to load configuration value for t10 <envTestStruct10.F>",
		},
		{
			name:       "bool invalid syntax",
			controller: t11,
			setup: func() {
				os.Setenv("t11", "xyz")
			},
			err: " ",
		},
		{
			name:       "int invalid syntax",
			controller: t12,
			setup: func() {
				os.Setenv("t12", "xyz")
			},
			err: " ",
		},
		{
			name:       "unsupported tag value",
			controller: t13,
			setup: func() {
				os.Setenv("t13", "xyz")
			},
			err: "unsupported tag value ",
		},
		{
			name:       "default config value",
			controller: t14,
			setup:      func() {},
			expected: &envTestStruct14{
				B: 42,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup()
			}
			_, err := resolveDependency(getEntry(&test.controller), registry)
			if test.err == "" {
				if err != nil {
					t.Fail()
				}
				if !reflect.DeepEqual(test.controller, test.expected) {
					t.Fail()
				}
			} else {
				if err != nil && err.Error() != test.err {
					t.Fatal(err.Error())
				}
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	type args struct {
		cfgKey string
		cmdArgs   []string
	}
	tests := []struct {
		name  string
		args      args
		wantValue string
		wantOk    bool
	}{
		{name: "Key found", args: args{cfgKey: "mytest", cmdArgs: []string{"--mytest", "Hello"}}, wantValue: "Hello", wantOk: true},
		{name: "Key not found", args: args{cfgKey: "mytest"}, wantValue: "", wantOk: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupArgs := os.Args
			os.Args = append(os.Args, tt.args.cmdArgs...)
			got, got1 := getConfig(tt.args.cfgKey)
			os.Args = backupArgs
			if got != tt.wantValue {
				t.Errorf("getConfig() got = %v, want %v", got, tt.wantValue)
			}
			if got1 != tt.wantOk {
				t.Errorf("getConfig() got1 = %v, want %v", got1, tt.wantOk)
			}
		})
	}
}

type envTestStruct1 struct {
	a int
	B int
	C string `boot:"config,key:t1"`
	d interface{}
	e []interface{}
}

func (t envTestStruct1) Init() {}

func (t envTestStruct1) do1() {}

type envTestStruct2 struct {
	a int
	B int
	C string `boot:"config,key:t2"`
	d interface{}
	e []interface{}
}

func (t envTestStruct2) do2() {}

func (t envTestStruct2) Init() {}

type envTestStruct3 struct {
	a int
	B int
	C string `boot:"config,key:t3,panic"`
	d interface{}
	e []interface{}
}

func (t envTestStruct3) Init() {}

type envTestStruct4 struct {
	a int
	B int
	C string `bo-ot:"config,key:t1,panic"`
	d interface{}
	e []interface{}
}

func (t envTestStruct4) Init() {}

type envTestStruct5 struct {
	a int
	B int
	C string `boot:"wi-re,key:t3,panic"`
	d interface{}
	e []interface{}
}

func (t envTestStruct5) Init() {}

type envTestStruct6 struct {
	a int
	B int
	C string `boot:"config"`
	d interface{}
	e []interface{}
}

func (t envTestStruct6) Init() {}

type envTestStruct7 struct {
	a int
	B int `boot:"config,key:t7"`
	C string
	d interface{}
	e []interface{}
}

func (t envTestStruct7) Init() {}

type envTestStruct8 struct {
	a int
	B int `boot:"config,key:t8,panic"`
	C string
	d interface{}
	e []interface{}
}

func (t envTestStruct8) Init() {}

type envTestStruct9 struct {
	a int
	B int
	C string
	d interface{}
	e []interface{}
	F bool `boot:"config,key:t9,panic"`
}

func (t envTestStruct9) Init() {}

type envTestStruct10 struct {
	a int
	B int
	C string
	d interface{}
	e []interface{}
	F bool `boot:"config,key:t10,panic"`
}

func (t envTestStruct10) Init() {}

type envTestStruct11 struct {
	a int
	B int
	C string
	d interface{}
	e []interface{}
	F bool `boot:"config,key:t11"`
}

func (t envTestStruct11) Init() {}

type envTestStruct12 struct {
	a int
	B int `boot:"config,key:t12"`
	C string
	d interface{}
	e []interface{}
	F bool
}

func (t envTestStruct12) Init() {}

type envTestStruct13 struct {
	a int
	B int `boot:"config,key"`
	C string
	d interface{}
	e []interface{}
	F bool
}

func (t envTestStruct13) Init() {}

type envTestStruct14 struct {
	a int
	B int `boot:"config,key:UNKNOWN,default:42"`
	C string
	d interface{}
	e []interface{}
	F bool
}

func (t envTestStruct14) Init() {}
