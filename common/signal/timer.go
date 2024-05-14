package signal

import (
	"context"
	"sync"
	"time"
	"log"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/task"
)

type ActivityUpdater interface {
	Update()
}

type ActivityTimer struct {
	sync.RWMutex
	updated   chan struct{}
	checkTask *task.Periodic
	onTimeout func()
}

func (t *ActivityTimer) Update() {
	select {
	case t.updated <- struct{}{}:
	default:
	}

	log.Println("ActivityTimer updating finished")
}

func (t *ActivityTimer) check() error {
	if t.checkTask != nil {
		log.Println("checking with interval", t.checkTask.Interval)
	}

	select {
	case <-t.updated:
		log.Println("ActivityTimer checking, activity detected")
	default:
		log.Println("ActivityTimer checking, no activity detected, calling finish")
		t.finish()
	}

	log.Println("ActivityTimer checking finished")
	return nil
}

func (t *ActivityTimer) finish() {
	t.Lock()
	defer t.Unlock()

	log.Println("ActivityTimer finishing")

	if t.onTimeout != nil {
		log.Println("ActivityTimer calling onTimeout")
		t.onTimeout()
		t.onTimeout = nil
	}
	if t.checkTask != nil {
		log.Println("ActivityTimer closing checkTask", t.checkTask.Interval)
		t.checkTask.Close()
		t.checkTask = nil
	}

	log.Println("ActivityTimer finished")
}

func (t *ActivityTimer) SetTimeout(timeout time.Duration) {
	log.Println("ActivityTimer SetTimeout", timeout)

	if timeout == 0 {
		t.finish()
		return
	}

	checkTask := &task.Periodic{
		Interval: timeout,
		Execute:  t.check,
	}

	t.Lock()

	if t.checkTask != nil {
		t.checkTask.Close()
	}
	t.checkTask = checkTask
	t.Unlock()
	t.Update()
	common.Must(checkTask.Start())

	log.Println("ActivityTimer SetTimeout finished")
}

func CancelAfterInactivity(ctx context.Context, cancel context.CancelFunc, timeout time.Duration) *ActivityTimer {
	timer := &ActivityTimer{
		updated:   make(chan struct{}, 1),
		onTimeout: cancel,
	}
	log.Println("ActivityTimer CancelAfterInactivity", timeout)
	timer.SetTimeout(timeout)
	return timer
}
