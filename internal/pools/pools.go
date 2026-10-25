// Package pools reads the state of every ZFS pool through zpool.
package pools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Member is one vdev or leaf of a pool's configuration tree.
type Member struct {
	Name   string `json:"name"`
	Depth  int    `json:"depth"`
	State  string `json:"state"`
	Read   int64  `json:"read_errors"`
	Write  int64  `json:"write_errors"`
	Cksum  int64  `json:"cksum_errors"`
	Note   string `json:"note"`
}

// Pool is the state of one pool, keyed by name.
type Pool struct {
	Name          string   `json:"name"`
	Health        string   `json:"health"`
	SizeBytes     int64    `json:"size_bytes"`
	AllocBytes    int64    `json:"alloc_bytes"`
	FreeBytes     int64    `json:"free_bytes"`
	CapacityPct   int64    `json:"capacity_pct"`
	Fragmentation int64    `json:"fragmentation_pct"`
	Scan          string   `json:"scan"`
	Status        string   `json:"status"`
	Errors        string   `json:"errors"`
	Members       []Member `json:"members"`
}

// Runner runs zpool with the configured path and privilege command.
type Runner struct {
	Zpool     string
	Privilege string
}

func (r Runner) run(ctx context.Context, args ...string) ([]byte, error) {
	var cmd *exec.Cmd
	if r.Privilege != "" {
		cmd = exec.CommandContext(ctx, r.Privilege, append([]string{r.Zpool}, args...)...)
	} else {
		cmd = exec.CommandContext(ctx, r.Zpool, args...)
	}
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("zpool %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	if err != nil {
		return nil, fmt.Errorf("zpool %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

func parseInt(field string) int64 {
	value, err := strconv.ParseInt(strings.TrimSuffix(field, "%"), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func (r Runner) list(ctx context.Context) (map[string]*Pool, []string, error) {
	out, err := r.run(ctx, "list", "-H", "-p", "-o", "name,size,alloc,free,cap,frag,health")
	if err != nil {
		return nil, nil, err
	}
	pools := map[string]*Pool{}
	order := []string{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 7 {
			continue
		}
		pool := &Pool{
			Name:          fields[0],
			SizeBytes:     parseInt(fields[1]),
			AllocBytes:    parseInt(fields[2]),
			FreeBytes:     parseInt(fields[3]),
			CapacityPct:   parseInt(fields[4]),
			Fragmentation: parseInt(fields[5]),
			Health:        fields[6],
			Members:       []Member{},
		}
		pools[pool.Name] = pool
		order = append(order, pool.Name)
	}
	return pools, order, nil
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func parseMember(line string, baseIndent int) (Member, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return Member{}, false
	}
	member := Member{
		Name:  fields[0],
		Depth: (indentOf(line) - baseIndent) / 2,
		State: fields[1],
	}
	if len(fields) >= 5 {
		member.Read = parseInt(fields[2])
		member.Write = parseInt(fields[3])
		member.Cksum = parseInt(fields[4])
		if len(fields) > 5 {
			member.Note = strings.Join(fields[5:], " ")
		}
	}
	return member, true
}

func (r Runner) status(ctx context.Context, pools map[string]*Pool) error {
	out, err := r.run(ctx, "status", "-p")
	if err != nil {
		return err
	}
	var current *Pool
	var section string
	var text strings.Builder
	configIndent := -1
	flush := func() {
		if current == nil {
			return
		}
		value := strings.TrimSpace(text.String())
		switch section {
		case "scan":
			current.Scan = value
		case "status":
			current.Status = value
		case "errors":
			current.Errors = value
		}
		text.Reset()
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if key, rest, found := strings.Cut(trimmed, ":"); found && indentOf(line) <= 2 && !strings.Contains(key, " ") {
			flush()
			section = key
			text.Reset()
			switch key {
			case "pool":
				current = pools[strings.TrimSpace(rest)]
				configIndent = -1
			case "config":
			default:
				text.WriteString(strings.TrimSpace(rest))
			}
			continue
		}
		if current == nil {
			continue
		}
		switch section {
		case "config":
			if strings.HasPrefix(trimmed, "NAME ") {
				configIndent = indentOf(line)
				continue
			}
			if configIndent < 0 {
				continue
			}
			if member, ok := parseMember(line, configIndent); ok {
				current.Members = append(current.Members, member)
			}
		default:
			text.WriteString(" ")
			text.WriteString(trimmed)
		}
	}
	flush()
	return nil
}

// Collect answers every pool in zpool list order.
func (r Runner) Collect(ctx context.Context) ([]Pool, error) {
	pools, order, err := r.list(ctx)
	if err != nil {
		return nil, err
	}
	if serr := r.status(ctx, pools); serr != nil {
		return nil, serr
	}
	out := make([]Pool, 0, len(order))
	for _, name := range order {
		out = append(out, *pools[name])
	}
	return out, nil
}
