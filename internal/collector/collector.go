// Package collector keeps the current report of every disk and pool and
// refreshes the part of it a trigger names: pools on a ZFS event, one disk on
// a SMART event, every disk on a device event, everything at start. Each
// refresh is handed to every subscriber.
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

// Collector holds the runners, the current report and its subscribers.
type Collector struct {
	Disks disks.Runner
	Pools pools.Runner

	runMu sync.Mutex

	mu      sync.Mutex
	current *Report
	subs    map[chan *Report]struct{}
}

func errText(err error) *string {
	if err == nil {
		return nil
	}
	text := err.Error()
	return &text
}

func emptyReport() *Report {
	return &Report{
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		Disks:       []disks.Disk{},
		Unreadable:  []disks.Unreadable{},
		Pools:       []pools.Pool{},
	}
}

// Current answers the latest report, an empty one before the first collection.
func (c *Collector) Current() *Report {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.current == nil {
		return emptyReport()
	}
	return c.current
}

// Subscribe answers a channel that carries every report after this call and
// the function that ends the subscription. A subscriber that has not read the
// previous report receives only the newest.
func (c *Collector) Subscribe() (reports <-chan *Report, unsubscribe func()) {
	ch := make(chan *Report, 1)
	c.mu.Lock()
	if c.subs == nil {
		c.subs = map[chan *Report]struct{}{}
	}
	c.subs[ch] = struct{}{}
	c.mu.Unlock()
	return ch, func() {
		c.mu.Lock()
		delete(c.subs, ch)
		c.mu.Unlock()
	}
}

func (c *Collector) publish(report *Report) {
	c.mu.Lock()
	c.current = report
	for ch := range c.subs {
		select {
		case <-ch:
		default:
		}
		ch <- report
	}
	c.mu.Unlock()
}

func (c *Collector) next() *Report {
	base := c.Current()
	report := *base
	report.CollectedAt = time.Now().UTC().Format(time.RFC3339)
	return &report
}

func (c *Collector) collectDisks(ctx context.Context, report *Report) {
	diskList, unreadable, err := c.Disks.Collect(ctx)
	report.DiskError = errText(err)
	if err != nil {
		return
	}
	report.Disks = diskList
	report.Unreadable = unreadable
	if report.Disks == nil {
		report.Disks = []disks.Disk{}
	}
	if report.Unreadable == nil {
		report.Unreadable = []disks.Unreadable{}
	}
}

func (c *Collector) collectPools(ctx context.Context, report *Report) {
	poolList, err := c.Pools.Collect(ctx)
	report.PoolError = errText(err)
	if err != nil {
		return
	}
	report.Pools = poolList
	if report.Pools == nil {
		report.Pools = []pools.Pool{}
	}
}

// CollectAll refreshes every disk and every pool.
func (c *Collector) CollectAll(ctx context.Context) *Report {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	report := c.next()
	c.collectDisks(ctx, report)
	c.collectPools(ctx, report)
	c.publish(report)
	return report
}

// CollectPools refreshes the pools only.
func (c *Collector) CollectPools(ctx context.Context) *Report {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	report := c.next()
	c.collectPools(ctx, report)
	c.publish(report)
	return report
}

// CollectDisks refreshes every disk only.
func (c *Collector) CollectDisks(ctx context.Context) *Report {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	report := c.next()
	c.collectDisks(ctx, report)
	c.publish(report)
	return report
}

// CollectDisk re-reads one device path; a device not in the current report
// refreshes every disk instead.
func (c *Collector) CollectDisk(ctx context.Context, device string) *Report {
	c.runMu.Lock()
	defer c.runMu.Unlock()
	report := c.next()
	index := -1
	for i := range report.Disks {
		if report.Disks[i].Device == device {
			index = i
			break
		}
	}
	if index < 0 {
		c.collectDisks(ctx, report)
		c.publish(report)
		return report
	}
	fresh := c.Disks.Read(ctx, device)
	list := make([]disks.Disk, len(report.Disks))
	copy(list, report.Disks)
	if fresh.Serial == "" {
		list = append(list[:index], list[index+1:]...)
		text := "no serial number"
		if fresh.Error != nil {
			text = *fresh.Error
		}
		report.Unreadable = append(append([]disks.Unreadable{}, report.Unreadable...), disks.Unreadable{Device: device, Error: text})
	} else {
		list[index] = fresh
	}
	report.Disks = list
	c.publish(report)
	return report
}
