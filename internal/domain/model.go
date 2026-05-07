package domain

type Memory struct {
	Total     int64
	Available int64
	Used      int64
}

type Process struct {
	Pid       int
	Ppid      int
	Rss       int
	CPU       float64
	Cmdline   string
	Username  string
	IsKthread bool
}

type Snapshot struct {
	CPU       float64
	Memory    Memory
	Processes []Process
}
