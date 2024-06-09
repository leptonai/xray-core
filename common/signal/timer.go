package signal

import (
	"context"
	"sync"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/log"
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

	log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer updating finished")
}

func (t *ActivityTimer) check() error {
	if t.checkTask != nil {
		log.RecordWithSeverity(log.Severity_Debug, "checking with interval", t.checkTask.Interval)
	}

	select {
	case <-t.updated:
		log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer checking, activity detected")
	default:
		log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer checking, no activity detected, calling finish")
		t.finish()
	}

	log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer checking finished")
	return nil
}

func (t *ActivityTimer) finish() {
	t.Lock()
	defer t.Unlock()

	log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer finishing")

	if t.onTimeout != nil {
		log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer calling onTimeout")
		t.onTimeout()
		t.onTimeout = nil
	}
	if t.checkTask != nil {
		log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer closing checkTask", t.checkTask.Interval)
		t.checkTask.Close()
		t.checkTask = nil
	}

	log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer finished")
}

func (t *ActivityTimer) SetTimeout(timeout time.Duration) {
	log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer SetTimeout", timeout)

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

	log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer SetTimeout finished")
}

func CancelAfterInactivity(ctx context.Context, cancel context.CancelFunc, timeout time.Duration) *ActivityTimer {
	timer := &ActivityTimer{
		updated:   make(chan struct{}, 1),
		onTimeout: cancel,
	}
	log.RecordWithSeverity(log.Severity_Debug, "ActivityTimer CancelAfterInactivity", timeout)
	timer.SetTimeout(timeout)
	return timer
}
