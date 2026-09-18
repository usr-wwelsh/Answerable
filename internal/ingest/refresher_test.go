package ingest

import (
	"testing"
	"time"
)

type fakeTicker struct {
	ch chan time.Time
}

func (f fakeTicker) C() <-chan time.Time { return f.ch }
func (f fakeTicker) Stop()               {}

func TestRefresherCallsRefreshOnTick(t *testing.T) {
	ch := make(chan time.Time)
	restore := newTicker
	newTicker = func(time.Duration) ticker { return fakeTicker{ch: ch} }
	defer func() { newTicker = restore }()

	called := make(chan struct{}, 1)
	r := NewRefresher(time.Minute, func() { called <- struct{}{} })
	r.Start()
	defer r.Stop()

	ch <- time.Now()

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("refresh func was not called after tick")
	}
}

func TestRefresherStopHaltsFurtherCalls(t *testing.T) {
	ch := make(chan time.Time)
	restore := newTicker
	newTicker = func(time.Duration) ticker { return fakeTicker{ch: ch} }
	defer func() { newTicker = restore }()

	called := make(chan struct{}, 2)
	r := NewRefresher(time.Minute, func() { called <- struct{}{} })
	r.Start()

	// Synchronize on one real tick so we know the goroutine is parked in
	// its select loop before we stop it.
	ch <- time.Now()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("refresh func was not called after first tick")
	}

	r.Stop()

	select {
	case ch <- time.Now():
	default:
	}

	select {
	case <-called:
		t.Fatal("refresh func was called after Stop")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRefresherZeroIntervalDoesNotStart(t *testing.T) {
	called := make(chan struct{}, 1)
	r := NewRefresher(0, func() { called <- struct{}{} })
	r.Start()
	defer r.Stop()

	select {
	case <-called:
		t.Fatal("refresh func was called with zero interval")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRefresherRefreshNowCallsImmediately(t *testing.T) {
	called := make(chan struct{}, 1)
	r := NewRefresher(time.Minute, func() { called <- struct{}{} })

	r.RefreshNow()

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("RefreshNow did not call refresh func")
	}
}
