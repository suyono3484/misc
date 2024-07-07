package graceful

/*
Copyright 2024 Suyono

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"
	"sync"
	"testing"
	"time"
)

type testGraceful struct {
	wait         time.Duration
	ready        chan int
	completeChan chan int
}

func (t *testGraceful) RunGracefully(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	t.ready <- 1
	<-ctx.Done()
	t.completeChan <- 1

	if t.wait > 0 {
		time.Sleep(t.wait)
	}
}

func TestStartServer(t *testing.T) {
	triggerFunc := func(t *testing.T, ready, completeChan chan int, cancel context.CancelFunc) {
		<-ready // service ready signal
		<-ready // service ready signal
		//t.Log("services are ready")

		cancel() // simulate shutdown signal
		//t.Log("shutdown initiated")

		<-completeChan // service completion signal
		<-completeChan // service completion signal
		//t.Log("shutdown complete")
	}

	cases := []struct {
		name        string
		deadline    time.Duration
		services    []*testGraceful
		wantTimeout bool
	}{
		{
			name:     "normal",
			deadline: 200 * time.Millisecond,
			services: []*testGraceful{
				&testGraceful{},
				&testGraceful{},
			},
			wantTimeout: false,
		},
		{
			name:     "timeout",
			deadline: 200 * time.Millisecond,
			services: []*testGraceful{
				&testGraceful{},
				&testGraceful{
					wait: 210 * time.Millisecond,
				},
			},
			wantTimeout: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ready := make(chan int)
			completeChan := make(chan int)

			svcs := make([]Service, 0, len(tc.services))
			for _, s := range tc.services {
				s.ready = ready
				s.completeChan = completeChan
				svcs = append(svcs, s)
			}

			startTime := time.Now()
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(tc.deadline))
			go triggerFunc(t, ready, completeChan, cancel)
			StartServer(ctx, time.Second, svcs...)
			stopTime := time.Since(startTime)
			if (stopTime >= tc.deadline) != tc.wantTimeout {
				t.Errorf("got timeout %v, want %v", stopTime, tc.wantTimeout)
			}

			close(ready)
			close(completeChan)
		})
	}
}
