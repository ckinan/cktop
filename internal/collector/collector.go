package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ckinan/sysmon/internal"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/process"
)

type Snapshot struct {
	CPU       float64
	Ram       internal.Ram
	Processes []internal.Process
}

func Start(ctx context.Context, interval time.Duration) <-chan Snapshot {
	ch := make(chan Snapshot, 1)

	go func() {
		defer close(ch) // signal consumers when goroutine exits
		ticker := time.NewTicker(interval)
		defer ticker.Stop() // vmlinuz

		// procCache lives here: only this goroutine touches it (no mutex needed)
		procCache := make(map[int32]*process.Process)

		// Collect immediately on start, don't wait for first tick
		if snap, err := collect(procCache); err == nil {
			ch <- snap
		}

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled (e.g. user pressed q, or main() exited)
				// Return immediately, defer close(ch) will run
				return
			case <-ticker.C:
				// Ticker fired: collect metrics and send Snapshot
				snap, err := collect(procCache)
				if err != nil {
					// skip this tick on error (e.g. a /proc read failed)
					slog.Warn("error reading resources", "error", err)
					continue
				}
				select {
				case ch <- snap:
					// snapshot sent successfully
				default:
					// Consumer hasn't read the previous snapshot yet
					// Drop this tick rather than blocking the gorouting
					// This keeps the collector running even if the UI is slow
				}

			}
		}
	}()
	return ch // return immediately, goroutine runs in background
}

func collect(procCache map[int32]*process.Process) (Snapshot, error) {
	ram, err := internal.GetRam()
	if err != nil {
		return Snapshot{}, err
	}

	cpuPcts, err := cpu.Percent(0, false)
	cpu := 0.0
	if err == nil && len(cpuPcts) > 0 {
		cpu = cpuPcts[0]
	}

	// Get the current list of live processes (lightweight handles)
	fresh, err := process.Processes()
	if err != nil {
		return Snapshot{}, err
	}

	// Build set of live PIDs; add new PIDs to cache
	livePIDs := make(map[int32]bool, len(fresh))
	for _, p := range fresh {
		livePIDs[p.Pid] = true
		if _, ok := procCache[p.Pid]; !ok {
			procCache[p.Pid] = p // first time seeing this PID
		}
	}

	// Evict handles for processes that no longer exist
	for pid := range procCache {
		if !livePIDs[pid] {
			delete(procCache, pid)
		}
	}

	// Fan-out using CACHED handles -> CPUPercent() has history from previous tick
	results := make(chan internal.Process, len(procCache))
	var wg sync.WaitGroup
	for _, p := range procCache {
		wg.Add(1)
		go func(p *process.Process) {
			defer wg.Done()
			if proc, err := internal.ReadProcess(p); err == nil {
				results <- proc
			}
		}(p)
	}
	wg.Wait()
	close(results)

	var processes []internal.Process
	for p := range results {
		processes = append(processes, p)
	}

	return Snapshot{
		CPU:       cpu,
		Ram:       ram,
		Processes: processes,
	}, nil
}
