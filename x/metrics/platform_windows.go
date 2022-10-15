package metrics

import (
	"github.com/shirou/gopsutil/process"
	"github.com/tychoish/birch"
)

const cpuTicks = 10000000.0

func marshalMemExtra(*process.MemoryInfoExStat) *birch.Element { return nil }
