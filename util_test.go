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
	"reflect"
	"testing"
)

type qualifiedNameComponent struct{}

func TestQualifiedName(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "struct",
			args: args{v: qualifiedNameComponent{}},
			want: "github.com/boot-go/boot/qualifiedNameComponent",
		},
		{
			name: "struct pointer",
			args: args{v: &qualifiedNameComponent{}},
			want: "github.com/boot-go/boot/qualifiedNameComponent",
		},
		{
			name: "string",
			args: args{v: ""},
			want: "string",
		},
		{
			name: "byte a.k.a uint8",
			args: args{v: byte(0)},
			want: "uint8",
		},
		{
			name: "int",
			args: args{v: 0},
			want: "int",
		},
		{
			name: "int64",
			args: args{v: int64(0)},
			want: "int64",
		},
		{
			name: "bool",
			args: args{v: true},
			want: "bool",
		},
		{
			name: "nil",
			args: args{v: nil},
			want: "nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QualifiedName(tt.args.v); got != tt.want {
				t.Errorf("QualifiedName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	type args struct {
		s     string
		sep   string
		quote string
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 bool
	}{
		{
			name: "simple",
			args: args{
				s:     "'hello,world'",
				sep:   ",",
				quote: "'",
			},
			want:  []string{"'hello,world'"},
			want1: true,
		},
		{
			name: "none",
			args: args{
				s:     "hello,world",
				sep:   ",",
				quote: "'",
			},
			want:  []string{"hello", "world"},
			want1: true,
		},
		{
			name: "two",
			args: args{
				s:     "'hello','world'",
				sep:   ",",
				quote: "'",
			},
			want:  []string{"'hello'", "'world'"},
			want1: true,
		},
		{
			name: "both",
			args: args{
				s:     "'hello,world','here'",
				sep:   ",",
				quote: "'",
			},
			want:  []string{"'hello,world'", "'here'"},
			want1: true,
		},
		{
			name: "failure_missing_quote",
			args: args{
				s:     "'hello,world,here",
				sep:   ",",
				quote: "'",
			},
			want:  nil,
			want1: false,
		},
		{
			name: "failure",
			args: args{
				s:     "hello,world,here'",
				sep:   ",",
				quote: "'",
			},
			want:  nil,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := Split(tt.args.s, tt.args.sep, tt.args.quote)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Split() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestLoggerMute(t *testing.T) {
	t.Run("mute test", func(t *testing.T) {
		Logger.Mute(Logger.Error)
		Logger.Mute(Logger.Warn)
		Logger.Mute(Logger.Info)
		Logger.Mute(Logger.Debug)
	})
}

func TestLoggerUnmute(t *testing.T) {
	t.Run("unmute test", func(t *testing.T) {
		Logger.Unmute(Logger.Error)
		Logger.Unmute(Logger.Warn)
		Logger.Unmute(Logger.Info)
		Logger.Unmute(Logger.Debug)
	})
}
