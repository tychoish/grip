package metrics

import "github.com/shirou/gopsutil/cpu"

var cpuTicks = cpu.ClocksPerSec
