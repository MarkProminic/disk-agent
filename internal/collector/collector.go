// Package collector runs the disk and pool collections for one report. A
// report is taken when a client asks for one; callers arriving while a
// collection runs receive that collection's result.
package collector

import (
	"context"
	"sync"
	"time"

	"github.com/MarkProminic/disk-agent/internal/disks"
	"github.com/MarkProminic/disk-agent/internal/pools"
)

// Report is the state of every disk and pool at one moment.
type Report struct {
	CollectedAt string             `json:"collected_at"`
	Disks       []disks.Disk       `json:"disks"`
	Unreadable  []disks.Unreadable `json:"unreadable"`
	Pools       []pools.Pool       `json:"pools"`
	DiskError   *string            `json:"disk_error"`
	PoolError   *string            `json:"pool_error"`
}

// Collector holds the runners and serializes collections.
type Collector struct {
	Disks disks.Runner
	Pools pools.Runner

	mu      sync.Mutex
	running chan struct{}
	last    *Report
}

func errText(err error) *string {
	if err == nil {
		return nil
	}
	text := err.Error()
	return &text
}

func (c *Collector) collect(ctx context.Context) *Report {
	report := &Report{CollectedAt: time.Now().UTC().Format(time.RFC3339)}
	diskList, unreadable, derr := c.Disks.Collect(ctx)
	report.Disks = diskList
	report.Unreadable = unreadable
	report.DiskError = errText(derr)
	poolList, perr := c.Pools.Collect(ctx)
	report.Pools = poolList
	report.PoolError = errText(perr)
	if report.Disks == nil {
		report.Disks = []disks.Disk{}
	}
	if report.Unreadable == nil {
		report.Unreadable = []disks.Unreadable{}
	}
	if report.Pools == nil {
		report.Pools = []pools.Pool{}
	}
	return report
}

// Report answers a fresh report, or the one a concurrent caller is already taking.
func (c *Collector) Report(ctx context.Context) *Report {
	c.mu.Lock()
	if c.running != nil {
		done := c.running
		c.mu.Unlock()
		<-done
		c.mu.Lock()
		report := c.last
		c.mu.Unlock()
		return report
	}
	done := make(chan struct{})
	c.running = done
	c.mu.Unlock()

	report := c.collect(ctx)

	c.mu.Lock()
	c.last = report
	c.running = nil
	c.mu.Unlock()
	close(done)
	return report
}
