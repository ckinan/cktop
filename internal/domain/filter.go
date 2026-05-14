package domain

import (
	"fmt"
	"strings"
)

// FilterProcesses returns the subset of procs whose fields contain query (case-insensitive).
// Returns procs unchanged if query is empty.
func FilterProcesses(procs []Process, query string) []Process {
	if query == "" {
		return procs
	}
	q := strings.ToLower(query)
	var out []Process
	for _, p := range procs {
		if strings.Contains(strings.ToLower(fmt.Sprintf("%d %d %s %s %d", p.Pid, p.Ppid, p.Username, p.Cmdline, p.Rss)), q) {
			out = append(out, p)
		}
	}
	return out
}
