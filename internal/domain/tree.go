package domain

// BuildChildren returns a map of pid -> list of child pids.
// Duplicate PIDs (gopsutil /proc race) are skipped.
func BuildChildren(procs []Process) map[int][]int {
	childrenByPid := make(map[int][]int)
	seen := make(map[int]bool, len(procs))
	for _, p := range procs {
		if seen[p.Pid] {
			continue
		}
		seen[p.Pid] = true
		childrenByPid[p.Ppid] = append(childrenByPid[p.Ppid], p.Pid)
	}
	return childrenByPid
}

// BuildParents returns the ancestor chain of selected, from immediate parent to root.
// The chain is ordered nearest-first: [parent, grandparent, ..., root].
func BuildParents(procs []Process, selected Process) []Process {
	pByPid := make(map[int]Process, len(procs))
	for _, p := range procs {
		pByPid[p.Pid] = p
	}

	var chain []Process
	currentPPID := selected.Ppid
	for currentPPID != 0 {
		p, ok := pByPid[currentPPID]
		if !ok {
			break
		}
		chain = append(chain, p)
		currentPPID = p.Ppid
	}
	return chain
}
