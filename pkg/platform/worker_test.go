package platform

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWorkerStopWaitsForExit(t *testing.T) {
	exited := make(chan struct{})
	worker := NewWorker(func(ctx context.Context) {
		<-ctx.Done()
		close(exited)
	})
	if err := worker.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	default:
		t.Fatal("Stop 返回时任务未退出")
	}
	if err := worker.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestWorkerConcurrentStartStop(t *testing.T) {
	for i := 0; i < 50; i++ {
		worker := NewWorker(func(ctx context.Context) { <-ctx.Done() })
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		var wg sync.WaitGroup
		failures := make(chan error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			err := worker.Start(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				failures <- err
			}
		}()
		go func() {
			defer wg.Done()
			if err := worker.Stop(ctx); err != nil {
				failures <- err
			}
		}()
		wg.Wait()
		close(failures)
		for err := range failures {
			t.Error(err)
		}
		cancel()
	}
}
