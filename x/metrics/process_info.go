package metrics

import (
	"fmt"
	"os"
	"sync"

	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"github.com/tychoish/birch"
	"github.com/tychoish/grip/message"
)

// ProcessInfo holds the data for per-process statistics (e.g. cpu,
// memory, io). The Process info composers produce messages in this
// form.
type ProcessInfo struct {
	Message string `json:"msg" bson:"msg"`
	Payload struct {
		Pid            int32                    `json:"pid" bson:"pid"`
		Parent         int32                    `json:"parentPid" bson:"parentPid"`
		Threads        int                      `json:"numThreads" bson:"numThreads"`
		Command        string                   `json:"command" bson:"command"`
		CPU            StatCPUTimes             `json:"cpu" bson:"cpu"`
		IoStat         process.IOCountersStat   `json:"io" bson:"io"`
		NetStat        []net.IOCountersStat     `json:"net" bson:"net"`
		Memory         process.MemoryInfoStat   `json:"mem" bson:"mem"`
		MemoryPlatform process.MemoryInfoExStat `json:"memExtra" bson:"memExtra"`
		Errors         []string                 `json:"errors" bson:"errors"`
	}

	message.Base `json:"metadata,omitempty" bson:"metadata,omitempty"`

	loggable bool
	rendered string
}

///////////////////////////////////////////////////////////////////////////
//
// Constructors
//
///////////////////////////////////////////////////////////////////////////

// CollectProcessInfo returns a populated ProcessInfo message.Composer
// instance for the specified pid.
func CollectProcessInfo(pid int32) message.Composer {
	return MakeProcessInfo(pid, "")
}

// CollectProcessInfoSelf returns a populated ProcessInfo message.Composer
// for the pid of the current process.
func CollectProcessInfoSelf() message.Composer {
	return MakeProcessInfo(int32(os.Getpid()), "")
}

// CollectProcessInfoSelfWithChildren returns a slice of populated
// ProcessInfo message.Composer instances for the current process and
// all children processes.
func CollectProcessInfoSelfWithChildren() []message.Composer {
	return CollectProcessInfoWithChildren(int32(os.Getpid()))
}

// CollectProcessInfoWithChildren returns a slice of populated
// ProcessInfo message.Composer instances for the process with the
// specified pid and all children processes for that process.
func CollectProcessInfoWithChildren(pid int32) []message.Composer {
	var results []message.Composer
	parent, err := process.NewProcess(pid)
	if err != nil {
		return results
	}

	parentMsg := &ProcessInfo{}
	parentMsg.loggable = true
	parentMsg.populate(parent)
	results = append(results, parentMsg)

	for _, child := range getChildrenRecursively(parent) {
		cm := &ProcessInfo{}
		cm.loggable = true
		cm.populate(child)
		results = append(results, cm)
	}

	return results
}

// CollectAllProcesses returns a slice of populated ProcessInfo
// message.Composer interfaces for all processes currently running on
// a system.
func CollectAllProcesses() []message.Composer {
	numThreads := 32
	procs, err := process.Processes()
	if err != nil {
		return []message.Composer{}
	}
	if len(procs) < numThreads {
		numThreads = len(procs)
	}

	results := []message.Composer{}
	procChan := make(chan *process.Process, len(procs))
	for _, p := range procs {
		procChan <- p
	}
	close(procChan)
	wg := sync.WaitGroup{}
	wg.Add(numThreads)
	infoChan := make(chan *ProcessInfo, len(procs))
	for i := 0; i < numThreads; i++ {
		go func() {
			defer wg.Done()
			for p := range procChan {
				cm := &ProcessInfo{}
				cm.loggable = true
				cm.populate(p)
				infoChan <- cm
			}
		}()
	}
	wg.Wait()
	close(infoChan)
	for p := range infoChan {
		results = append(results, p)
	}

	return results
}

func getChildrenRecursively(proc *process.Process) []*process.Process {
	var out []*process.Process

	children, err := proc.Children()
	if len(children) == 0 || err != nil {
		return out
	}

	for _, p := range children {
		out = append(out, p)
		out = append(out, getChildrenRecursively(p)...)
	}

	return out
}

// NewProcessInfo constructs a fully configured and populated
// Processinfo message.Composer instance for the specified process.
func MakeProcessInfo(pid int32, message string) message.Composer {
	p := &ProcessInfo{}
	p.Message = message
	p.Payload.Pid = pid

	proc, err := process.NewProcess(pid)
	p.saveError("process", err)
	if err != nil {
		return p
	}

	p.loggable = true
	p.populate(proc)

	return p
}

///////////////////////////////////////////////////////////////////////////
//
// message.Composer implementation
//
///////////////////////////////////////////////////////////////////////////

// Loggable returns true when the Processinfo structure has been
// populated.
func (p *ProcessInfo) Loggable() bool { return p.loggable }
func (*ProcessInfo) Structured() bool { return true }
func (*ProcessInfo) Schema() string   { return "procinfo.0" }

// Raw always returns the ProcessInfo object, however it will call the
// Collect method of the base operation first.
func (p *ProcessInfo) Raw() any { p.Collect(); return p }

// String returns a string representation of the message, lazily
// rendering the message, and caching it privately.
func (p *ProcessInfo) String() string {
	if p.rendered == "" {
		p.rendered = renderStatsString(p.Message, p.Payload)
	}

	return p.rendered
}

func (p *ProcessInfo) MarshalDocument() (*birch.Document, error) {
	proc := birch.DC.Elements(
		birch.EC.Int32("pid", p.Payload.Pid),
		birch.EC.Int32("parentPid", p.Payload.Parent),
		birch.EC.Int("threads", p.Payload.Threads),
		birch.EC.String("command", p.Payload.Command),
		birch.EC.SubDocument("cpu", marshalCPU(&p.Payload.CPU)),
		birch.EC.SubDocumentFromElements("io",
			birch.EC.Int64("readCount", int64(p.Payload.IoStat.ReadCount)),
			birch.EC.Int64("writeCount", int64(p.Payload.IoStat.WriteCount)),
			birch.EC.Int64("readBytes", int64(p.Payload.IoStat.ReadBytes)),
			birch.EC.Int64("writeBytes", int64(p.Payload.IoStat.WriteBytes))),
		birch.EC.SubDocumentFromElements("mem",
			birch.EC.Int64("rss", int64(p.Payload.Memory.RSS)),
			birch.EC.Int64("vms", int64(p.Payload.Memory.VMS)),
			birch.EC.Int64("hwm", int64(p.Payload.Memory.HWM)),
			birch.EC.Int64("data", int64(p.Payload.Memory.Data)),
			birch.EC.Int64("stack", int64(p.Payload.Memory.Stack)),
			birch.EC.Int64("locked", int64(p.Payload.Memory.Locked)),
			birch.EC.Int64("swap", int64(p.Payload.Memory.Swap))),
	)

	proc.AppendOmitEmpty(marshalMemExtra(&p.Payload.MemoryPlatform))
	na := birch.MakeArray(len(p.Payload.NetStat))

	for _, netstat := range p.Payload.NetStat {
		na.Append(birch.VC.Document(marshalNetStat(&netstat)))
	}

	proc.Append(birch.EC.Array("net", na))
	return proc, nil
}

///////////////////////////////////////////////////////////////////////////
//
// Internal Methods for collecting data
//
///////////////////////////////////////////////////////////////////////////

func (p *ProcessInfo) populate(proc *process.Process) {
	var err error

	if p.Payload.Pid == 0 {
		p.Payload.Pid = proc.Pid
	}
	parentPid, err := proc.Ppid()
	p.saveError("parent_pid", err)
	if err == nil {
		p.Payload.Parent = parentPid
	}

	memInfo, err := proc.MemoryInfo()
	p.saveError("meminfo", err)
	if err == nil && memInfo != nil {
		p.Payload.Memory = *memInfo
	}

	memInfoEx, err := proc.MemoryInfoEx()
	p.saveError("meminfo_extended", err)
	if err == nil && memInfoEx != nil {
		p.Payload.MemoryPlatform = *memInfoEx
	}

	threads, err := proc.NumThreads()
	p.Payload.Threads = int(threads)
	p.saveError("num_threads", err)

	p.Payload.NetStat, err = proc.NetIOCounters(false)
	p.saveError("netstat", err)

	p.Payload.Command, err = proc.Cmdline()
	p.saveError("cmd args", err)

	cpuTimes, err := proc.Times()
	p.saveError("cpu_times", err)
	if err == nil && cpuTimes != nil {
		p.Payload.CPU = convertCPUTimes(*cpuTimes)
	}

	ioStat, err := proc.IOCounters()
	p.saveError("iostat", err)
	if err == nil && ioStat != nil {
		p.Payload.IoStat = *ioStat
	}
}

func (p *ProcessInfo) saveError(stat string, err error) {
	if shouldSaveError(err) {
		p.Payload.Errors = append(p.Payload.Errors, fmt.Sprintf("%s: %v", stat, err))
	}
}
