package metrics

import (
	"github.com/shirou/gopsutil/net"
	"github.com/tychoish/birch"
)

func marshalNetStat(netstat *net.IOCountersStat) *birch.Document {
	return birch.DC.Elements(
		birch.EC.String("name", netstat.Name),
		birch.EC.Int64("bytesSent", int64(netstat.BytesSent)),
		birch.EC.Int64("bytesRecv", int64(netstat.BytesRecv)),
		birch.EC.Int64("packetsSent", int64(netstat.PacketsSent)),
		birch.EC.Int64("packetsRecv", int64(netstat.PacketsRecv)),
		birch.EC.Int64("errin", int64(netstat.Errin)),
		birch.EC.Int64("errout", int64(netstat.Errout)),
		birch.EC.Int64("dropin", int64(netstat.Dropin)),
		birch.EC.Int64("dropout", int64(netstat.Dropout)),
		birch.EC.Int64("fifoin", int64(netstat.Fifoin)),
		birch.EC.Int64("fifoout", int64(netstat.Fifoout)))
}

func marshalCPU(cpu *StatCPUTimes) *birch.Document {
	return birch.DC.Elements(
		birch.EC.Int64("user", cpu.User),
		birch.EC.Int64("system", cpu.System),
		birch.EC.Int64("idle", cpu.Idle),
		birch.EC.Int64("nice", cpu.Nice),
		birch.EC.Int64("iowait", cpu.Iowait),
		birch.EC.Int64("irq", cpu.Irq),
		birch.EC.Int64("softirq", cpu.Softirq),
		birch.EC.Int64("steal", cpu.Steal),
		birch.EC.Int64("guest", cpu.Guest),
		birch.EC.Int64("guestNice", cpu.GuestNice))
}
