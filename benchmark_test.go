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
	"fmt"
	"testing"
	"time"
)

const (
	events = 100000

	receivers = 1000

	referenceTime = 35

	maxRetries = 5
)

type IncrementEvent struct {
	value int
}

type testReceiverComponent struct {
	EventBus EventBus `boot:"wire"`
	id       int
	count    int
}

func (t *testReceiverComponent) Init() error {
	t.count = 0
	err := t.EventBus.Subscribe(func(e IncrementEvent) {
		t.count = e.value
	})
	if err != nil {
		return err
	}
	return nil
}

type testSenderComponent struct {
	EventBus EventBus `boot:"wire"`
}

func (t *testSenderComponent) Init() error {
	return nil
}

func (t *testSenderComponent) Start() error {
	for i := 0; i < events; i++ {
		err := t.EventBus.Publish(IncrementEvent{
			value: i,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *testSenderComponent) Stop() error {
	return nil
}

func BenchmarkEventBus_Publish(b *testing.B) {
	success := false
	for try := 1; try < maxRetries+1; try++ {
		if success {
			break
		}
		fmt.Printf("run       : %v\n", try)
		startTime := time.Now()
		s := NewSession(UnitTestFlag)
		for i := 0; i < receivers; i++ {
			err := s.RegisterName(fmt.Sprintf("receiver-%d", i), func() Component {
				return &testReceiverComponent{
					id: i,
				}
			})
			if err != nil {
				b.Fatal(err)
			}
		}
		err := s.Register(func() Component {
			return &testSenderComponent{}
		})
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		err = s.Go()
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		relative := referenceTime / duration.Seconds() * 100
		eventRate := events / duration.Seconds()
		fmt.Printf("reference : %vs\n", referenceTime)
		fmt.Printf("events    : %v\n", events)
		fmt.Printf("receivers : %v\n", receivers)
		fmt.Printf("reference : %vs\n", referenceTime)
		fmt.Printf("duration  : %v\n", duration)
		fmt.Printf("result    : %.1f%%\n", relative)
		fmt.Printf("event rate: %.1f/s\n", eventRate)
		if duration.Seconds() <= referenceTime {
			success = true
		} else {
			fmt.Printf("insufficient performance - retrying...\n")
		}
	}
	if !success {
		b.Fatal("Benchmark failed")
	}
}
