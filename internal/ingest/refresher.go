package ingest

import "time"

type ticker interface {
	C() <-chan time.Time
	Stop()
}

type realTicker struct{ t *time.Ticker }

func (r realTicker) C() <-chan time.Time { return r.t.C }
func (r realTicker) Stop()               { r.t.Stop() }

var newTicker = func(d time.Duration) ticker { return realTicker{time.NewTicker(d)} }

// Refresher calls refresh on a fixed interval until Stop is called. An
// interval <= 0 disables periodic refresh (manual RefreshNow still works).
type Refresher struct {
	interval time.Duration
	refresh  func()
	stop     chan struct{}
}

func NewRefresher(interval time.Duration, refresh func()) *Refresher {
	return &Refresher{interval: interval, refresh: refresh, stop: make(chan struct{})}
}

func (r *Refresher) Start() {
	if r.interval <= 0 {
		return
	}

	go func() {
		t := newTicker(r.interval)
		defer t.Stop()
		for {
			select {
			case <-t.C():
				r.refresh()
			case <-r.stop:
				return
			}
		}
	}()
}

func (r *Refresher) Stop() {
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
}

func (r *Refresher) RefreshNow() {
	r.refresh()
}
