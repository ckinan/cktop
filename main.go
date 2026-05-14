package main

import (
	"context"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ckinan/cktop/internal/adapters/gopsutil"
	"github.com/ckinan/cktop/internal/domain"
	"github.com/ckinan/cktop/internal/infra"
	"github.com/ckinan/cktop/internal/ui"
)

// main has one job: wire the graph and start the program
// all dependencies are constructed here and injected to the domain
func main() {
	// context.WithCancel gives us a cancel function to stop the collector cleanly.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // ensure goroutine is always stopped when main() exits

	// Adapters / repositories
	memReader := gopsutil.GopsutilMemoryReader{}
	procReader := gopsutil.NewGopsutilProcessReader()
	cpuReader := gopsutil.GopsutilCPUReader{}

	// Domain (pure functions)
	collector := domain.NewCollector(memReader, procReader, cpuReader)

	// Infra: how collectors run
	// start collector - returns a channel immediately, goroutine runs in background
	snapshotCh := infra.Start(ctx, collector, 1*time.Second)

	// UI
	p := tea.NewProgram(ui.New(snapshotCh), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("error running TUI", "err", err)
	}
}
