package metrics

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/tychoish/birch"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// SystemInfo is a type that implements message.Composer but also
// collects system-wide resource utilization statistics about memory,
// CPU, and network use, along with an optional message.
type SystemInfo struct {
	Message      string                `json:"message" bson:"message"`
	CPU          StatCPUTimes          `json:"cpu" bson:"cpu"`
	CPUPercent   float64               `json:"cpu_percent" bson:"cpu_percent"`
	NumCPU       int                   `json:"num_cpus" bson:"num_cpus"`
	VMStat       mem.VirtualMemoryStat `json:"vmstat" bson:"vmstat"`
	NetStat      net.IOCountersStat    `json:"netstat" bson:"netstat"`
	Partitions   []disk.PartitionStat  `json:"partitions" bson:"partitions"`
	Usage        []disk.UsageStat      `json:"usage" bson:"usage"`
	IOStat       []disk.IOCountersStat `json:"iostat" bson:"iostat"`
	Errors       []string              `json:"errors" bson:"errors"`
	message.Base `json:"metadata,omitempty" bson:"metadata,omitempty"`
	loggable     bool
	rendered     string
}

// StatCPUTimes provides a mirror of gopsutil/cpu.TimesStat with
// integers rather than floats.
type StatCPUTimes struct {
	User      int64 `json:"user" bson:"user"`
	System    int64 `json:"system" bson:"system"`
	Idle      int64 `json:"idle" bson:"idle"`
	Nice      int64 `json:"nice" bson:"nice"`
	Iowait    int64 `json:"iowait" bson:"iowait"`
	Irq       int64 `json:"irq" bson:"irq"`
	Softirq   int64 `json:"softirq" bson:"softirq"`
	Steal     int64 `json:"steal" bson:"steal"`
	Guest     int64 `json:"guest" bson:"guest"`
	GuestNice int64 `json:"guestNice" bson:"guestNice"`
}

func convertCPUTimes(in cpu.TimesStat) StatCPUTimes {
	return StatCPUTimes{
		User:      int64(in.User * cpuTicks),
		System:    int64(in.System * cpuTicks),
		Idle:      int64(in.Idle * cpuTicks),
		Nice:      int64(in.Nice * cpuTicks),
		Iowait:    int64(in.Iowait * cpuTicks),
		Irq:       int64(in.Irq * cpuTicks),
		Softirq:   int64(in.Softirq * cpuTicks),
		Steal:     int64(in.Steal * cpuTicks),
		Guest:     int64(in.Guest * cpuTicks),
		GuestNice: int64(in.GuestNice * cpuTicks),
	}
}

// CollectSystemInfo returns a populated SystemInfo object,
// without a message.
func CollectSystemInfo() message.Composer {
	return NewSystemInfo(level.Trace, "")
}

// MakeSystemInfo builds a populated SystemInfo object with the
// specified message.
func MakeSystemInfo(message string) message.Composer {
	return NewSystemInfo(level.Info, message)
}

// NewSystemInfo returns a fully configured and populated SystemInfo
// object.
func NewSystemInfo(priority level.Priority, message string) message.Composer {
	var err error
	s := &SystemInfo{
		Message: message,
		NumCPU:  runtime.NumCPU(),
	}

	if err = s.SetPriority(priority); err != nil {
		s.Errors = append(s.Errors, err.Error())
		return s
	}

	s.loggable = true

	times, err := cpu.Times(false)
	s.saveError("cpu_times", err)
	if err == nil && len(times) > 0 {
		// since we're not storing per-core information,
		// there's only one thing we care about in this struct
		s.CPU = convertCPUTimes(times[0])
	}
	percent, err := cpu.Percent(0, false)
	if err != nil {
		s.saveError("cpu_times", err)
	} else {
		s.CPUPercent = percent[0]
	}

	vmstat, err := mem.VirtualMemory()
	s.saveError("vmstat", err)
	if err == nil && vmstat != nil {
		s.VMStat = *vmstat
		s.VMStat.UsedPercent = 0.0
	}

	netstat, err := net.IOCounters(false)
	s.saveError("netstat", err)
	if err == nil && len(netstat) > 0 {
		s.NetStat = netstat[0]
	}

	partitions, err := disk.Partitions(true)
	s.saveError("disk_part", err)

	if err == nil {
		var u *disk.UsageStat
		for _, p := range partitions {
			u, err = disk.Usage(p.Mountpoint)
			s.saveError("partition", err)
			if err != nil {
				continue
			}
			u.UsedPercent = 0.0
			u.InodesUsedPercent = 0.0

			s.Usage = append(s.Usage, *u)
		}

		s.Partitions = partitions
	}

	iostatMap, err := disk.IOCounters()
	s.saveError("iostat", err)
	for _, stat := range iostatMap {
		s.IOStat = append(s.IOStat, stat)
	}

	return s
}

// Loggable returns true when the Processinfo structure has been
// populated.
func (s *SystemInfo) Loggable() bool { return s.loggable }
func (*SystemInfo) Structured() bool { return true }
func (*SystemInfo) Schema() string   { return "sysinfo.0" }

// Raw always returns the SystemInfo object.
func (s *SystemInfo) Raw() any { return s }

// String returns a string representation of the message, lazily
// rendering the message, and caching it privately.
func (s *SystemInfo) String() string {
	if s.rendered == "" {
		s.rendered = renderStatsString(s.Message, s)
	}

	return s.rendered
}

func (s *SystemInfo) saveError(stat string, err error) {
	if shouldSaveError(err) {
		s.Errors = append(s.Errors, fmt.Sprintf("%s: %v", stat, err))
	}
}

func (s *SystemInfo) MarshalDocument() (*birch.Document, error) {
	sys := birch.DC.Elements(
		birch.EC.Int("num_cpu", s.NumCPU),
		birch.EC.Double("cpu_percent", s.CPUPercent),
		birch.EC.SubDocument("cpu", marshalCPU(&s.CPU)),
		birch.EC.SubDocumentFromElements("vmstat",
			birch.EC.Int64("total", int64(s.VMStat.Total)),
			birch.EC.Int64("available", int64(s.VMStat.Available)),
			birch.EC.Int64("used", int64(s.VMStat.Used)),
			birch.EC.Int64("usedPercent", int64(s.VMStat.UsedPercent)),
			birch.EC.Int64("free", int64(s.VMStat.Free)),
			birch.EC.Int64("active", int64(s.VMStat.Active)),
			birch.EC.Int64("inactive", int64(s.VMStat.Inactive)),
			birch.EC.Int64("wired", int64(s.VMStat.Wired)),
			birch.EC.Int64("laundry", int64(s.VMStat.Laundry)),
			birch.EC.Int64("buffers", int64(s.VMStat.Buffers)),
			birch.EC.Int64("cached", int64(s.VMStat.Cached)),
			birch.EC.Int64("writeback", int64(s.VMStat.Writeback)),
			birch.EC.Int64("dirty", int64(s.VMStat.Dirty)),
			birch.EC.Int64("writebacktmp", int64(s.VMStat.WritebackTmp)),
			birch.EC.Int64("shared", int64(s.VMStat.Shared)),
			birch.EC.Int64("slab", int64(s.VMStat.Slab)),
			birch.EC.Int64("sreclaimable", int64(s.VMStat.SReclaimable)),
			birch.EC.Int64("sunreclaim", int64(s.VMStat.SUnreclaim)),
			birch.EC.Int64("pagetables", int64(s.VMStat.PageTables)),
			birch.EC.Int64("swapcached", int64(s.VMStat.SwapCached)),
			birch.EC.Int64("commitlimit", int64(s.VMStat.CommitLimit)),
			birch.EC.Int64("commitedas", int64(s.VMStat.CommittedAS)),
			birch.EC.Int64("hightotal", int64(s.VMStat.HighTotal)),
			birch.EC.Int64("highfree", int64(s.VMStat.HighFree)),
			birch.EC.Int64("lowtotal", int64(s.VMStat.LowTotal)),
			birch.EC.Int64("lowfree", int64(s.VMStat.LowFree)),
			birch.EC.Int64("swaptotal", int64(s.VMStat.SwapTotal)),
			birch.EC.Int64("swapfree", int64(s.VMStat.SwapFree)),
			birch.EC.Int64("mapped", int64(s.VMStat.Mapped)),
			birch.EC.Int64("vmalloctotal", int64(s.VMStat.VMallocTotal)),
			birch.EC.Int64("vmallocused", int64(s.VMStat.VMallocUsed)),
			birch.EC.Int64("vmallocchunk", int64(s.VMStat.VMallocChunk)),
			birch.EC.Int64("hugepagestotal", int64(s.VMStat.HugePagesTotal)),
			birch.EC.Int64("hugepagesfree", int64(s.VMStat.HugePagesFree)),
			birch.EC.Int64("hugepagessize", int64(s.VMStat.HugePageSize))),
		birch.EC.SubDocument("netstat", marshalNetStat(&s.NetStat)))
	{
		ua := birch.MakeArray(len(s.Usage))
		for _, usage := range s.Usage {
			ua.Append(birch.VC.DocumentFromElements(
				birch.EC.String("path", usage.Path),
				birch.EC.String("fstype", usage.Fstype),
				birch.EC.Int64("total", int64(usage.Total)),
				birch.EC.Int64("free", int64(usage.Free)),
				birch.EC.Int64("used", int64(usage.Used)),
				birch.EC.Double("usedPercent", usage.UsedPercent),
				birch.EC.Int64("inodesTotal", int64(usage.InodesTotal)),
				birch.EC.Int64("inodesFree", int64(usage.InodesFree)),
				birch.EC.Double("inodesUsedPercent", usage.InodesUsedPercent)))
		}
		sys.Append(birch.EC.Array("usage", ua))
	}
	{
		ioa := birch.MakeArray(len(s.IOStat))
		for _, iostat := range s.IOStat {
			ioa.Append(birch.VC.DocumentFromElements(
				birch.EC.String("name", iostat.Name),
				birch.EC.String("serialNumber", iostat.SerialNumber),
				birch.EC.String("label", iostat.Label),
				birch.EC.Int64("readCount", int64(iostat.ReadCount)),
				birch.EC.Int64("mergedReadCount", int64(iostat.MergedReadCount)),
				birch.EC.Int64("writeCount", int64(iostat.WriteCount)),
				birch.EC.Int64("mergedWriteCount", int64(iostat.MergedWriteCount)),
				birch.EC.Int64("readBytes", int64(iostat.ReadBytes)),
				birch.EC.Int64("writeBytes", int64(iostat.WriteBytes)),
				birch.EC.Int64("readTime", int64(iostat.ReadTime)),
				birch.EC.Int64("writeTime", int64(iostat.WriteTime)),
				birch.EC.Int64("iopsInProgress", int64(iostat.IopsInProgress)),
				birch.EC.Int64("ioTime", int64(iostat.IoTime)),
				birch.EC.Int64("weightedIO", int64(iostat.WeightedIO)),
			))
		}
		sys.Append(birch.EC.Array("iostat", ioa))
	}
	{
		parts := birch.MakeArray(len(s.Partitions))
		for _, part := range s.Partitions {
			parts.Append(birch.VC.DocumentFromElements(
				birch.EC.String("device", part.Device),
				birch.EC.String("mountpoint", part.Mountpoint),
				birch.EC.String("fstype", part.Fstype),
				birch.EC.String("opts", part.Opts),
			))
		}
		sys.Append(birch.EC.Array("partitions", parts))
	}

	return sys, nil
}

// helper function
func shouldSaveError(err error) bool {
	return err != nil && err.Error() != "not implemented yet"
}

func renderStatsString(msg string, data any) string {
	out, err := json.Marshal(data)
	if err != nil {
		return msg
	}

	if msg == "" {
		return string(out)
	}

	return fmt.Sprintf("%s:\n%s", msg, string(out))
}
