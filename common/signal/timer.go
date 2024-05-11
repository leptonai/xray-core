package signal

import (
	"context"
	"sync"
	"time"

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
	log.Println("ActivityTimer updating")

	select {
	case t.updated <- struct{}{}:
		log.Println("ActivityTimer updated")
	default:
		log.Println("ActivityTimer already updated, skipping")
	}

	log.Println("ActivityTimer updating finished")
}

func (t *ActivityTimer) check() error {
	log.Println("ActivityTimer checking")

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
		log.Println("ActivityTimer closing checkTask")
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
		log.Println("ActivityTimer SetTimeout, closing existing checkTask")
		t.checkTask.Close()
	}
	t.checkTask = checkTask
	t.Unlock()
	t.Update()

	log.Println("ActivityTimer SetTimeout, starting checkTask")
	common.Must(checkTask.Start())

	log.Println("ActivityTimer SetTimeout finished")
}

func CancelAfterInactivity(ctx context.Context, cancel context.CancelFunc, timeout time.Duration) *ActivityTimer {
	timer := &ActivityTimer{
		updated:   make(chan struct{}, 1),
		onTimeout: cancel,
	}
	timer.SetTimeout(timeout)
	return timer
}
