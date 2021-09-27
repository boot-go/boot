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
	"reflect"
	"testing"
)

func TestResolveDependency(t *testing.T) {
	type args struct {
		resolveEntry *entry
		registry     *registry
	}
	tests := []struct {
		name        string
		args        args
		wantEntries []*entry
		wantErr     bool
	}{
		{name: "component already created", args: struct {
			resolveEntry *entry
			registry     *registry
		}{resolveEntry: &entry{
			component: nil,
			state:     Started,
			name:      DefaultName,
		}, registry: newRegistry()}, wantEntries: nil, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEntries, err := resolveDependency(tt.args.resolveEntry, tt.args.registry)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveDependency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotEntries, tt.wantEntries) {
				t.Errorf("resolveDependency() gotEntries = %v, want %v", gotEntries, tt.wantEntries)
			}
		})
	}
}

func TestParseStructTag(t *testing.T) {
	type args struct {
		tagValue string
	}
	tests := []struct {
		name  string
		args  args
		want  *tag
		want1 bool
	}{
		{
			name:  "wrong amount of sub-tokens",
			args:  args{tagValue: ",::::"},
			want:  nil,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := parseStructTag(tt.args.tagValue)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStructTag() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseStructTag() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

type testerComponentOne struct{}

func (t *testerComponentOne) Init() {}

type testerComponentTwo struct {
	One *testerComponentOne `boot:"wire"`
}

func (t *testerComponentTwo) Init() {}

func TestTesterWithMultipleTestComponents(t *testing.T) {
	err := Test(&testerComponentOne{}, &testerComponentTwo{})
	if err != nil {
		t.Errorf("Test failed: %s", err.Error())
	}
}
