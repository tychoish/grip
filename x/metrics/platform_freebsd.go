package metrics

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
	"github.com/tychoish/birch"
)

var cpuTicks = cpu.ClocksPerSec

func marshalMemExtra(*process.MemoryInfoExStat) *birch.Element { return nil }
