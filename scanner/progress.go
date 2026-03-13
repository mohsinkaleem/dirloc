package scanner

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

// Progress displays a live file counter on stderr for large scans.
type Progress struct {
	count int64 // accessed atomically
	start time.Time
	stop  chan struct{}
	done  sync.WaitGroup
}

// NewProgress creates a Progress reporter. Returns nil if stderr is not a
// terminal (piped/redirected), so callers can safely pass nil elsewhere.
func NewProgress() *Progress {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}
	return &Progress{
		start: time.Now(),
		stop:  make(chan struct{}),
	}
}

// Inc increments the scanned-file counter. Safe for concurrent use.
func (p *Progress) Inc() {
	if p == nil {
		return
	}
	atomic.AddInt64(&p.count, 1)
}

// Count returns the current scanned-file count.
func (p *Progress) Count() int64 {
	if p == nil {
		return 0
	}
	return atomic.LoadInt64(&p.count)
}

// Start begins the periodic status line on stderr.
func (p *Progress) Start() {
	if p == nil {
		return
	}
	p.done.Add(1)
	go func() {
		defer p.done.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n := atomic.LoadInt64(&p.count)
				elapsed := time.Since(p.start)
				fmt.Fprintf(os.Stderr, "\r\033[KScanning... %d files [%s]", n, fmtElapsed(elapsed))
			case <-p.stop:
				fmt.Fprint(os.Stderr, "\r\033[K") // clear the line
				return
			}
		}
	}()
}

// Stop ends the progress display and waits for the goroutine to finish.
func (p *Progress) Stop() {
	if p == nil {
		return
	}
	close(p.stop)
	p.done.Wait()
}

func fmtElapsed(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
