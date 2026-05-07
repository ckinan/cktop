package ui

import (
	"fmt"

	"github.com/ckinan/cktop/internal/util"
)

func (m Model) View() string {
	if m.showDetail {
		header := fmt.Sprintf(
			"PID: %d | PPID: %d | User: %s | CPU: %.2f%% | RSS: %s\nCmdLine: %s",
			m.frozenProc.Pid,
			m.frozenProc.Ppid,
			m.frozenProc.Username,
			m.frozenProc.CPU,
			util.HumanBytes(int64(m.frozenProc.Rss)),
			m.frozenProc.Cmdline,
		)
		footer := m.footerView("[enter]details [/]search [H]kthreads [F9]kill [q]back")
		return header + "\n" + m.tableDetail.View() + "\n\n" + footer
	}
	memPct := 0.0
	if m.memory.Total > 0 {
		memPct = float64(m.memory.Used) * 100.0 / float64(m.memory.Total)
	}
	header := fmt.Sprintf(
		"CPU: %.2f%%\nMem: %s / %s (%.2f%%)\n",
		m.CPU,
		util.HumanBytes(m.memory.Used),
		util.HumanBytes(m.memory.Total),
		memPct,
	)
	footer := m.footerView("sort: [C]cpu [M]rss [P]pid [L]cmdline | [enter]details [/]search [H]kthreads [F9]kill [q]quit")
	return header + "\n" + m.table.View() + "\n\n" + footer
}

func (m Model) footerView(hints string) string {
	if m.killPending {
		return fmt.Sprintf("Kill PID %d: [t]SIGTERM [k]SIGKILL [esc]cancel", m.killPID)
	}
	if m.killMsg != "" {
		return fmt.Sprintf("%s | %s", m.killMsg, hints)
	}
	if m.filterActive {
		return m.filter.View()
	}
	if m.filter.Value() != "" {
		return fmt.Sprintf("/%s  [esc]clear | %s", m.filter.Value(), hints)
	}
	return hints
}
