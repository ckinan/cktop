package domain_test

import (
	"testing"

	"github.com/ckinan/cktop/internal/domain"
)

// TestBuildChildren_DuplicatePIDs guards against a known gopsutil behavior:
// /proc can return the same PID twice during a race. BuildChildren must deduplicate
// so the process tree does not double-count children.
func TestBuildChildren_DuplicatePIDs(t *testing.T) {
	procs := []domain.Process{
		{Pid: 1, Ppid: 0},
		{Pid: 2, Ppid: 1},
		{Pid: 2, Ppid: 1}, // duplicate from /proc race
	}

	children := domain.BuildChildren(procs)

	if len(children[1]) != 1 {
		t.Errorf("expected 1 child of PID 1, got %d: %v", len(children[1]), children[1])
	}
}

// TestBuildParents_MissingParent guards against an infinite loop or panic when
// a process references a PPID that no longer exists in the snapshot
// (e.g. the parent exited between reads).
func TestBuildParents_MissingParent(t *testing.T) {
	procs := []domain.Process{
		{Pid: 42, Ppid: 99}, // parent 99 is not in the list
	}
	selected := domain.Process{Pid: 42, Ppid: 99}

	parents := domain.BuildParents(procs, selected)

	if len(parents) != 0 {
		t.Errorf("expected empty parent chain, got %d entries", len(parents))
	}
}
