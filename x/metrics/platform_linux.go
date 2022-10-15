package metrics

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
	"github.com/tychoish/birch"
)

var cpuTicks = cpu.ClocksPerSec

func marshalMemExtra(mem *process.MemoryInfoExStat) *birch.Element {
	if mem == nil {
		return nil
	}
	return birch.EC.SubDocumentFromElements("memExtra",
		birch.EC.Int64("rss", int64(mem.RSS)),
		birch.EC.Int64("vms", int64(mem.VMS)),
		birch.EC.Int64("shared", int64(mem.Shared)),
		birch.EC.Int64("text", int64(mem.Text)),
		birch.EC.Int64("lib", int64(mem.Lib)),
		birch.EC.Int64("data", int64(mem.Data)),
		birch.EC.Int64("dirty", int64(mem.Dirty)))
}
