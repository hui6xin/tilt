package engine

import (
	"sync"

	"github.com/windmilleng/tilt/internal/watch"
)

type fakeMultiWatcher struct {
	events chan watch.FileEvent
	errors chan error

	mu         sync.Mutex
	subs       []chan watch.FileEvent
	subsErrors []chan error
}

func newFakeMultiWatcher() *fakeMultiWatcher {
	r := &fakeMultiWatcher{events: make(chan watch.FileEvent), errors: make(chan error)}
	go r.loop()
	return r
}

func (w *fakeMultiWatcher) newSub() (watch.Notify, error) {
	subCh := make(chan watch.FileEvent)
	errorCh := make(chan error)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subs = append(w.subs, subCh)
	w.subsErrors = append(w.subsErrors, errorCh)
	return newFakeWatcher(subCh, errorCh), nil
}

func (w *fakeMultiWatcher) getSubs() []chan watch.FileEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.subs
}

func (w *fakeMultiWatcher) getSubErrors() []chan error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.subsErrors
}

func (w *fakeMultiWatcher) loop() {
	for {
		select {
		case e, ok := <-w.events:
			if !ok {
				for _, sub := range w.getSubs() {
					close(sub)
				}
				return
			}
			for _, sub := range w.getSubs() {
				sub <- e
			}
		case e, ok := <-w.errors:
			if !ok {
				for _, sub := range w.getSubErrors() {
					close(sub)
				}
				return
			}
			for _, sub := range w.getSubErrors() {
				sub <- e
			}
		}
	}
}

type fakeWatcher struct {
	inboundCh  chan watch.FileEvent
	outboundCh chan watch.FileEvent
	errorCh    chan error
}

func newFakeWatcher(inboundCh chan watch.FileEvent, errorCh chan error) *fakeWatcher {
	r := &fakeWatcher{inboundCh: inboundCh, outboundCh: make(chan watch.FileEvent), errorCh: errorCh}
	go r.loop()

	return r
}

func (w *fakeWatcher) Add(name string) error {
	return nil
}

func (w *fakeWatcher) Close() error {
	return nil
}

func (w *fakeWatcher) Errors() chan error {
	return w.errorCh
}

func (w *fakeWatcher) Events() chan watch.FileEvent {
	return w.outboundCh
}

func (w *fakeWatcher) loop() {
	var q []watch.FileEvent
	for {
		var outboundCh chan watch.FileEvent
		var outboundE watch.FileEvent
		if len(q) > 0 {
			outboundCh, outboundE = w.outboundCh, q[0]
		}

		select {
		case e, ok := <-w.inboundCh:
			if !ok {
				close(w.outboundCh)
				return
			}
			q = append(q, e)
		case outboundCh <- outboundE:
			q = q[1:]
		}
	}
}

var _ watch.Notify = &fakeWatcher{}