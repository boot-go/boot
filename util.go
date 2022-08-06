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
	"io"
	"log"
	"os"
	"reflect"
	gort "runtime"
	"strings"
)

// QualifiedName returns the full name of a struct, function or a simple name of a primitive.
func QualifiedName(v any) string {
	t := reflect.TypeOf(v)
	if t != nil {
		switch t.Kind() { //nolint:exhaustive // all others are covered by default
		case reflect.Ptr:
			return t.Elem().PkgPath() + "/" + t.Elem().Name()
		case reflect.Func:
			return gort.FuncForPC(reflect.ValueOf(v).Pointer()).Name()
		default:
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

// Split returns the tokens separated by sep and ignores content in the quotes. If the parsing is okay, the bool return value will be true.
// Unfortunately, the regex can't be used to split the string, because Go has limitations.
func Split(s, sep, quote string) ([]string, bool) {
	var result []string
	tokens := strings.Split(s, sep)
	summarizedToken := ""
	summarizing := false
	for _, token := range tokens {
		count := len(strings.Split(token, quote)) - 1
		if count == 1 || count%2 != 0 {
			if summarizing {
				// summarizing completed
				summarizing = false
				summarizedToken += sep + token
			} else {
				// summarizing started
				summarizing = true
				summarizedToken = token
			}
		} else {
			if summarizing {
				// summarizing ongoing
				summarizedToken += sep + token
			} else {
				// no summarizing at all
				summarizedToken = token
			}
		}
		if !summarizing {
			result = append(result, summarizedToken)
			summarizedToken = ""
		}
	}
	if summarizedToken == "" {
		return result, true
	} else {
		return nil, false
	}
}

// logger contains multiple loggers which can be seen as different
// log levels
type logger struct {
	Debug *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
}

// Mute will mute the provided logger.
func (l logger) Mute(logger *log.Logger) {
	logger.SetOutput(io.Discard)
}

// Unmute will unmute the provided logger.
func (l logger) Unmute(logger *log.Logger) {
	logger.SetOutput(os.Stdout)
}

var (
	// Logger contains a debug, info, warning and error logger, which is used for fine-grained log
	// output. Every logger can be muted or unmuted separately.
	// e.g. Logger.Unmute(Logger.Debug)
	Logger logger
)

func init() {
	Logger.Debug = log.New(os.Stdout, "boot.debug ", log.LstdFlags|log.Lmsgprefix)
	Logger.Info = log.New(os.Stdout, "boot.info ", log.LstdFlags|log.Lmsgprefix)
	Logger.Warn = log.New(os.Stdout, "boot.warn ", log.LstdFlags|log.Lmsgprefix)
	Logger.Error = log.New(os.Stdout, "boot.error ", log.LstdFlags|log.Lmsgprefix)
	// default is to mute the debug logger
	Logger.Mute(Logger.Debug)
}
