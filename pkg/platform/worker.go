package platform

import (
	"context"
	"sync"
)

// Worker 管理后台任务；mu 保护启动、取消和停止状态，done 只在任务退出时关闭。
type Worker struct {
	run     func(context.Context)
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
	stopped bool
}

// NewWorker 创建由应用生命周期管理的后台任务。
func NewWorker(run func(context.Context)) *Worker {
	return &Worker{
		run:  run,
		done: make(chan struct{}),
	}
}

// Start 只启动一次后台任务，已经停止的任务不能重新启动。
func (w *Worker) Start(context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return context.Canceled
	}
	if w.started {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.started = true
	go func() {
		defer close(w.done)
		w.run(ctx)
	}()
	return nil
}

// Stop 取消任务并等待退出；可与 Start 并发且可重复调用。
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		if w.started {
			w.cancel()
		} else {
			close(w.done)
		}
	}
	w.mu.Unlock()
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
