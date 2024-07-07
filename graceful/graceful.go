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
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Service interface {
	RunGracefully(ctx context.Context, wg *sync.WaitGroup)
}

func StartServer(ctx context.Context, waiterTimeOut time.Duration, services ...Service) {
	if ctx == nil {
		ctx = context.Background()
	}

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM) // for Windows, see notes in https://pkg.go.dev/os/signal#hdr-Windows
	wg := &sync.WaitGroup{}

	// Start goroutines here, each goroutine should accept wg and signalCtx for graceful shutdown
	for _, service := range services {
		wg.Add(1)
		go service.RunGracefully(signalCtx, wg)
	}

	<-signalCtx.Done()
	stop()

	waiterCtx, waiterComplete := context.WithTimeout(context.Background(), waiterTimeOut)
	go waiter(wg, waiterComplete)
	<-waiterCtx.Done()
}

func waiter(wg *sync.WaitGroup, cancel context.CancelFunc) {
	wg.Wait()
	cancel()
}
