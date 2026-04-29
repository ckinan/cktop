package internal

import (
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

type Process struct {
	Pid      int // process id
	Ppid     int // parent process id
	Rss      int // bytes
	CPU      float64
	Cmdline  string
	Username string
}

func ReadProcess(p *process.Process) (Process, error) {
	ppid, err := p.Ppid()
	if err != nil {
		return Process{}, err
	}
	mem, err := p.MemoryInfo()
	if err != nil {
		return Process{}, err
	}
	// cmdline will error on kernel threads (they do not have cmdline)
	// so let's not evaluate the errors for them
	cmdline, _ := p.Cmdline()
	// do not show the full path, only the executable and the args
	if cmdline != "" {
		parts := strings.SplitN(cmdline, " ", 2)
		parts[0] = filepath.Base(parts[0])
		cmdline = strings.Join(parts, " ")
	}
	username, err := p.Username()
	if err != nil {
		return Process{}, err
	}
	cpu, err := p.Percent(0)
	if err != nil {
		cpu = 0 // not fatal
	}
	return Process{
		Pid:      int(p.Pid),
		Ppid:     int(ppid),
		Rss:      int(mem.RSS),
		CPU:      cpu,
		Cmdline:  cmdline,
		Username: username,
	}, nil
}
