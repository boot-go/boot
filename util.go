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
	"io/ioutil"
	"log"
	"os"
	"reflect"
	gort "runtime"
)

// QualifiedName returns the full name of a struct, function or a simple name of a primitive.
func QualifiedName(v interface{}) string {
	t := reflect.TypeOf(v)
	if t != nil {
		if t.Kind() == reflect.Ptr {
			return t.Elem().PkgPath() + "/" + t.Elem().Name()
		} else if t.Kind() == reflect.Func {
			return gort.FuncForPC(reflect.ValueOf(v).Pointer()).Name()
		} else {
			pkg := t.PkgPath()
			if pkg != "" {
				pkg += "/"
			}
			return pkg + reflect.TypeOf(v).Name()
		}
	} else {
		return "nil"
	}
}

var (
	// Logger contains a debug, info, warning and error logger, which is used for fine-grained log
	// output. Every logger can be muted or unmuted separately.
	Logger struct {
		Debug *log.Logger
		Info  *log.Logger
		Warn  *log.Logger
		Error *log.Logger
	}
)

func init() {
	Logger.Debug = log.New(os.Stdout, "boot.debug ", log.LstdFlags|log.Lmsgprefix)
	Logger.Debug.SetOutput(ioutil.Discard)
	Logger.Info = log.New(os.Stdout, "boot..info ", log.LstdFlags|log.Lmsgprefix)
	Logger.Warn = log.New(os.Stdout, "boot..warn ", log.LstdFlags|log.Lmsgprefix)
	Logger.Error = log.New(os.Stdout, "boot.error ", log.LstdFlags|log.Lmsgprefix)
}
