package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ckinan/cktop/internal/domain"
	"github.com/ckinan/cktop/internal/util"
)

type snapshotMsg domain.Snapshot

func waitForSnapshot(ch <-chan domain.Snapshot) tea.Cmd {
	return func() tea.Msg {
		snap, ok := <-ch
		if !ok {
			return nil
		}
		return snapshotMsg(snap)
	}
}

func calcDir(showDir bool, sortDesc bool) string {
	if !showDir {
		return ""
	}
	if sortDesc == true {
		return " ▼"
	}
	return " ▲"
}

func (m *Model) applySort() {
	cmdW := max(20, m.width-colPIDWidth-colPPIDWidth-colUserWidth-colCPUWidth-colRSSWidth)

	m.table.SetColumns([]table.Column{
		{Title: "PID" + calcDir(m.sortBy == SortByPID, m.sortDesc), Width: colPIDWidth},
		{Title: "PPID", Width: colPPIDWidth},
		{Title: "User", Width: colUserWidth},
		{Title: "CPU%" + calcDir(m.sortBy == SortByCPU, m.sortDesc), Width: colCPUWidth},
		{Title: "RSS" + calcDir(m.sortBy == SortByRSS, m.sortDesc), Width: colRSSWidth},
		{Title: "CmdLine" + calcDir(m.sortBy == SortByCmdLine, m.sortDesc), Width: cmdW},
	})

	var sorted []domain.Process
	procs := domain.FilterProcesses(m.procs, m.filter.Value())
	if !m.showKThreads {
		filtered := procs[:0]
		for _, p := range procs {
			if !p.IsKthread {
				filtered = append(filtered, p)
			}
		}
		procs = filtered
	}
	switch m.sortBy {
	case SortByRSS:
		sorted = util.SortBy(procs, func(p domain.Process) int { return p.Rss }, m.sortDesc)
	case SortByCPU:
		sorted = util.SortBy(procs, func(p domain.Process) float64 { return p.CPU }, m.sortDesc)
	case SortByPID:
		sorted = util.SortBy(procs, func(p domain.Process) int { return p.Pid }, m.sortDesc)
	case SortByPPID:
		sorted = util.SortBy(procs, func(p domain.Process) int { return p.Ppid }, m.sortDesc)
	case SortByCmdLine:
		sorted = util.SortBy(procs, func(p domain.Process) string { return p.Cmdline }, m.sortDesc)
	}

	rows := make([]table.Row, len(sorted))
	for i, p := range sorted {
		rows[i] = table.Row{
			fmt.Sprintf("%d", p.Pid),
			fmt.Sprintf("%d", p.Ppid),
			p.Username,
			fmt.Sprintf("%.2f%%", p.CPU),
			util.HumanBytes(int64(p.Rss)),
			p.Cmdline,
		}
	}
	m.table.SetRows(rows)
}

// appendTreeRows recursively builds table rows for the descendants of pid.
// prefix carries the vertical-bar context from parent levels so connectors line up correctly.
// Children are sorted by PID for a stable layout across live refreshes.
func appendTreeRows(pid int, pByPid map[int]domain.Process, childrenByPid map[int][]int, prefix string, rows []table.Row, pids []int) ([]table.Row, []int) {
	children := make([]int, len(childrenByPid[pid]))
	copy(children, childrenByPid[pid])
	sort.Ints(children)
	for i, childPid := range children {
		isLast := i == len(children)-1
		connector, nextPrefix := "├─ ", prefix+"│  "
		if isLast {
			connector, nextPrefix = "└─ ", prefix+"   "
		}
		p := pByPid[childPid]
		rows = append(rows, table.Row{fmt.Sprintf("%s%s[pid:%d | cpu:%.2f%% | rss:%s] %s", prefix, connector, p.Pid, p.CPU, util.HumanBytes(int64(p.Rss)), p.Cmdline)})
		pids = append(pids, p.Pid)
		rows, pids = appendTreeRows(childPid, pByPid, childrenByPid, nextPrefix, rows, pids)
	}
	return rows, pids
}

func buildTreeRows(procs []domain.Process, selected domain.Process) ([]table.Row, []int) {
	pByPid := make(map[int]domain.Process, len(procs))
	for _, p := range procs {
		pByPid[p.Pid] = p
	}
	childrenByPid := domain.BuildChildren(procs)
	parents := domain.BuildParents(procs, selected)

	var rows []table.Row
	var pids []int

	// ancestors: root to immediate parent (single chain, so "└─" is always correct)
	for depth, i := 0, len(parents)-1; i >= 0; i, depth = i-1, depth+1 {
		p := parents[i]
		if depth == 0 {
			rows = append(rows, table.Row{fmt.Sprintf("[pid:%d | cpu:%.2f%% | rss:%s] %s", p.Pid, p.CPU, util.HumanBytes(int64(p.Rss)), p.Cmdline)})
		} else {
			rows = append(rows, table.Row{fmt.Sprintf("%s└─ [pid:%d | cpu:%.2f%% | rss:%s] %s", strings.Repeat("   ", depth-1), p.Pid, p.CPU, util.HumanBytes(int64(p.Rss)), p.Cmdline)})
		}
		pids = append(pids, p.Pid)
	}

	// selected process
	depth := len(parents)
	if depth == 0 {
		rows = append(rows, table.Row{fmt.Sprintf("[pid:%d | cpu:%.2f%% | rss:%s] %s", selected.Pid, selected.CPU, util.HumanBytes(int64(selected.Rss)), selected.Cmdline)})
	} else {
		rows = append(rows, table.Row{fmt.Sprintf("%s└─ [pid:%d | cpu:%.2f%% | rss:%s] %s", strings.Repeat("   ", depth-1), selected.Pid, selected.CPU, util.HumanBytes(int64(selected.Rss)), selected.Cmdline)})
	}
	pids = append(pids, selected.Pid)

	// children subtree
	childPrefix := strings.Repeat("   ", depth)
	rows, pids = appendTreeRows(selected.Pid, pByPid, childrenByPid, childPrefix, rows, pids)

	return rows, pids
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case snapshotMsg:
		snap := domain.Snapshot(msg)
		m.CPU = msg.CPU
		m.memory = msg.Memory
		wasEmpty := len(m.procs) == 0 // first data arrival?
		m.procs = snap.Processes
		m.applySort()
		if wasEmpty {
			m.table.GotoTop()
		}
		if m.showDetail {
			found := false
			for _, p := range m.procs {
				if p.Pid == m.detailProc.Pid {
					m.detailProc = p
					found = true
					break
				}
			}
			if found {
				m.detailProcDead = false
				m.openDetailView()
			} else {
				m.detailProcDead = true
			}
		}
		return m, waitForSnapshot(m.snapCh)
	case tea.KeyMsg:
		m.killMsg = ""
		if m.killPending {
			switch msg.String() {
			case "t":
				if err := syscall.Kill(m.killPID, syscall.SIGTERM); err != nil {
					m.killMsg = fmt.Sprintf("SIGTERM failed: %s", err)
				} else {
					m.killMsg = fmt.Sprintf("sent SIGTERM to PID %d", m.killPID)
				}
				m.killPending = false
			case "k":
				if err := syscall.Kill(m.killPID, syscall.SIGKILL); err != nil {
					m.killMsg = fmt.Sprintf("SIGKILL failed: %s", err)
				} else {
					m.killMsg = fmt.Sprintf("sent SIGKILL to PID %d", m.killPID)
				}
				m.killPending = false
			case "esc":
				m.killPending = false
			}
			return m, nil
		}
		if m.filterActive {
			switch msg.String() {
			case "enter":
				m.filterActive = false
				m.filter.Blur()
				if m.showDetail {
					m.openDetailView()
				} else {
					m.applySort()
				}
				return m, nil
			case "esc":
				m.filterActive = false
				m.filter.Blur()
				m.filter.SetValue("")
				if m.showDetail {
					m.openDetailView()
				} else {
					m.applySort()
				}
				return m, nil
			}
			var tiCmd tea.Cmd
			m.filter, tiCmd = m.filter.Update(msg)
			if m.showDetail {
				m.openDetailView()
			} else {
				m.applySort()
			}
			return m, tiCmd
		}
		if m.showDetail {
			switch msg.String() {
			case "/":
				m.filterActive = true
				m.filter.SetValue("")
				m.filter.Focus()
				return m, textinput.Blink
			case "esc":
				if m.filter.Value() != "" {
					m.filter.SetValue("")
					m.openDetailView()
				}
				return m, nil
			case "f9":
				cursor := m.tableDetail.Cursor()
				if cursor >= 0 && cursor < len(m.treeRowPIDs) {
					m.killPID = m.treeRowPIDs[cursor]
					m.killPending = true
				}
				return m, nil
			case "H":
				m.showKThreads = !m.showKThreads
				return m, nil
			case "q":
				m.showDetail = false
				m.filter.SetValue("")
				return m, nil
			case "enter":
				cursor := m.tableDetail.Cursor()
				if cursor >= 0 && cursor < len(m.treeRowPIDs) {
					pid := m.treeRowPIDs[cursor]
					for _, p := range m.procs {
						if p.Pid == pid {
							m.detailProc = p
							break
						}
					}
					m.detailProcDead = false
					m.treeRowPIDs = nil // reset cursor so openDetailView lands on the new process
					m.filter.SetValue("")
					m.openDetailView()
				}
				return m, nil
			}
			m.tableDetail, cmd = m.tableDetail.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "/":
			m.filterActive = true
			m.filter.SetValue("")
			m.filter.Focus()
			return m, textinput.Blink
		case "esc":
			if m.filter.Value() != "" {
				m.filter.SetValue("")
				m.applySort()
			}
			return m, nil
		case "f9":
			row := m.table.SelectedRow()
			if len(row) > 0 {
				pid, _ := strconv.Atoi(row[0])
				m.killPID = pid
				m.killPending = true
			}
			return m, nil
		case "H":
			m.showKThreads = !m.showKThreads
			m.applySort()
			return m, nil
		}
		prev := m.sortBy
		isSortKey := true
		switch msg.String() {
		case "enter":
			isSortKey = false

			selectedPID := m.table.SelectedRow()[0]
			selectedPIDint, _ := strconv.Atoi(selectedPID)
			for _, p := range m.procs {
				if p.Pid == selectedPIDint {
					m.detailProc = p
					break
				}
			}
			m.detailProcDead = false
			m.treeRowPIDs = nil // reset cursor so openDetailView lands on the selected process
			m.filter.SetValue("")
			m.openDetailView()
			m.showDetail = true
		case "M":
			m.sortBy = SortByRSS
		case "C":
			m.sortBy = SortByCPU
		case "P":
			m.sortBy = SortByPID
		case "L":
			m.sortBy = SortByCmdLine
		case "q":
			return m, tea.Quit
		default:
			isSortKey = false
		}
		if isSortKey {
			if m.sortBy == prev {
				// same key: toggle direction
				m.sortDesc = !m.sortDesc
			} else {
				// new field: reset to descending
				m.sortDesc = true
			}
			m.applySort()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.table.SetHeight(m.height - 4)       // 1 header + 1 blank + table + 1 blank + 1 footer
		m.tableDetail.SetHeight(m.height - 5) // 3 header lines + 1 blank + 1 footer
		m.applySort()

		return m, nil
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Model) openDetailView() {
	// Remember which PID is under the cursor so we can restore it after rebuild.
	var cursorPID int
	if c := m.tableDetail.Cursor(); c >= 0 && c < len(m.treeRowPIDs) {
		cursorPID = m.treeRowPIDs[c]
	}

	rows, pids := buildTreeRows(m.procs, m.detailProc)

	if q := m.filter.Value(); q != "" {
		q = strings.ToLower(q)
		var filteredRows []table.Row
		var filteredPIDs []int
		for i, r := range rows {
			if strings.Contains(strings.ToLower(r[0]), q) {
				filteredRows = append(filteredRows, r)
				filteredPIDs = append(filteredPIDs, pids[i])
			}
		}
		rows, pids = filteredRows, filteredPIDs
	}

	m.tableDetail.SetRows(rows)
	m.treeRowPIDs = pids
	// Restore cursor: prefer the previously focused row, fall back to detailProc.
	for i, pid := range pids {
		if pid == cursorPID {
			m.tableDetail.SetCursor(i)
			return
		}
	}
	for i, pid := range pids {
		if pid == m.detailProc.Pid {
			m.tableDetail.SetCursor(i)
			return
		}
	}
	m.tableDetail.SetCursor(0)
}
