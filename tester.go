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
	"os"
)

type testSession struct {
	*Session
}

// newTestSession creates a new session with one or more specific components within a unit test.
func newTestSession(mocks ...Component) *testSession {
	ts := &testSession{
		Session: NewSession(UnitTestFlag),
	}
	Logger.Debug.SetOutput(os.Stdout)
	// ignore error because it can't fail
	_ = ts.overrideTestComponent(mocks...)
	return ts
}

func (ts *testSession) overrideTestComponent(mocks ...Component) error {
	for _, mock := range mocks {
		mock := mock
		err := ts.register(DefaultName, func() Component {
			return mock
		}, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ts *testSession) registerTestComponent(mocks ...Component) error {
	for _, mock := range mocks {
		mock := mock
		err := ts.register(DefaultName, func() Component {
			return mock
		}, false)
		if err != nil {
			return err
		}
	}
	return nil
}
