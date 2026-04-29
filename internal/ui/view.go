package ui

import (
	"fmt"

	"github.com/ckinan/sysmon/internal"
)

func (m Model) View() string {
	header := fmt.Sprintf(
		"CPU: %.2f%%\nMem: %s / %s (%.2f%%)\n",
		m.CPU,
		internal.HumanBytes(m.ram.MemUsed),
		internal.HumanBytes(m.ram.MemTotal),
		float64(m.ram.MemUsed)*100.0/float64(m.ram.MemTotal),
	)
	footer := "[q] quit"
	return header + "\n" + m.table.View() + "\n\n" + footer
}
