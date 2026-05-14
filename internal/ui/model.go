package ui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ckinan/cktop/internal/domain"
)

const (
	colPIDWidth     = 8
	colPPIDWidth    = 8
	colUserWidth    = 10
	colCPUWidth     = 8
	colRSSWidth     = 10
	colCommandWidth = 40
)

type SortField int

const (
	SortByRSS SortField = iota // default: highest RSS first
	SortByCPU
	SortByPID
	SortByPPID
	SortByCmdLine
)

func (s SortField) String() string {
	switch s {
	case SortByRSS:
		return "RSS"
	case SortByCPU:
		return "CPU"
	case SortByPID:
		return "PID"
	case SortByPPID:
		return "PPID"
	case SortByCmdLine:
		return "CmdLine"
	default:
		return "?"
	}
}

// Model is the bubbletea model. It holds all UI state
type Model struct {
	snapCh      <-chan domain.Snapshot // read-only channel from the collect
	CPU         float64
	memory      domain.Memory
	procs       []domain.Process
	height      int // terminal height
	width       int
	table       table.Model
	sortBy      SortField
	sortDesc    bool
	tableDetail table.Model
	treeRowPIDs []int // maps tableDetail row index → PID
	// filter state (shared between both views)
	filter        textinput.Model
	filterActive  bool
	showKThreads  bool
	// kill state
	killPending bool
	killPID     int
	killMsg     string
	// fields for details view
	showDetail     bool
	detailProc     domain.Process
	detailProcDead bool
}

func New(ch <-chan domain.Snapshot) Model {
	// height: 24 is a safe fallback
	// frame is painted right after startup, so this default is almost never actually visible
	cols := []table.Column{
		{Title: "PID", Width: colPIDWidth},
		{Title: "PPID", Width: colPPIDWidth},
		{Title: "User", Width: colUserWidth},
		{Title: "CPU%", Width: colCPUWidth},
		{Title: "RSS", Width: colRSSWidth},
		{Title: "CmdLine", Width: colCommandWidth},
	}
	s := table.DefaultStyles()
	s.Selected = lipgloss.NewStyle().Reverse(true)
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true), // focused = keyboard nav (↑/↓) is active
		table.WithStyles(s),
	)

	fi := textinput.New()
	fi.Prompt = "/"
	fi.CharLimit = 64

	td := table.New(
		table.WithColumns([]table.Column{{Title: "", Width: 120}}),
		table.WithFocused(true),
		table.WithStyles(s),
	)
	return Model{
		snapCh:      ch,
		height:      24,
		table:       t,
		tableDetail: td,
		filter:      fi,
		sortBy:      SortByRSS,
		sortDesc:    true,
	}
}

func (m Model) Init() tea.Cmd {
	return waitForSnapshot(m.snapCh)
}
