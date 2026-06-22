package cf_cache_buster

import (
	"sync"
	"time"
)

type Debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	duration time.Duration
}

// NewDebouncer creates a new Debouncer instance.
func NewDebouncer(d time.Duration) *Debouncer {
	return &Debouncer{
		duration: d,
	}
}

// Run schedules the function f to be executed after the debounce duration.
// If Run is called again before the timer expires, the previous execution is canceled.
func (d *Debouncer) Run(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.duration, f)
}