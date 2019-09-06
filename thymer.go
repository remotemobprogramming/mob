package main

import (
	"math"
	"sync"
	"time"
)

type Thymer struct {
	wg       sync.WaitGroup
	duration time.Duration
	interval time.Duration
	closeCh  chan bool
	notifyCh chan ThymerNotification
}

type ThymerNotification struct {
	TimeLeft time.Duration
	PercLeft float64
	Done     bool
}

func NewThymer(duration, interval time.Duration) *Thymer {
	return &Thymer{
		duration: duration,
		interval: interval,
	}
}

func (t *Thymer) Start(notifyCh chan ThymerNotification) {
	t.closeCh = make(chan bool)
	t.notifyCh = notifyCh
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		defer close(t.notifyCh)
		t.starttick()
	}()
}

func (t *Thymer) starttick() {
	start := time.Now()
	for {

		timeLeft := t.duration - time.Since(start)
		if timeLeft < 0 {
			timeLeft = 0
		}
		t.notifyCh <- ThymerNotification{
			TimeLeft: timeLeft,
			PercLeft: math.Ceil(float64((timeLeft * 100) / t.duration)),
			Done:     (timeLeft <= 0),
		}
		if timeLeft <= 0 {
			return
		}
		select {
		case <-t.closeCh:
			return
		case <-time.After(t.interval):
		}
	}
}

func (t *Thymer) Stop() {
	close(t.closeCh)
	t.wg.Wait()
}

func (t *Thymer) Wait() {
	t.wg.Wait()
}
