package cgroups

type ThrottlingStat struct {
	Periods          uint64
	ThrottledPeriods uint64
	ThrottledTime    uint64
}
type CPUUsage struct {
	Total  uint64
	User   uint64
	Kernel uint64
	PerCPU []uint64
}
type CPUStat struct {
	Throttling *ThrottlingStat
	Usage      *CPUUsage
}
type BlkIOEntry struct {
	Op     string
	Device string
	Major  uint64
	Minor  uint64
	Value  uint64
}
type BlkIOStat struct {
	SectorsRecursive        []*BlkIOEntry
	IoServiceBytesRecursive []*BlkIOEntry
	IoServicedRecursive     []*BlkIOEntry
	IoServiceTimeRecursive  []*BlkIOEntry
	IoWaitTimeRecursive     []*BlkIOEntry
	IoMergedRecursive       []*BlkIOEntry
	IoTimeRecursive         []*BlkIOEntry
}
type CpuacctStat struct {
}
type CpusetStat struct {
}
type MemoryEntry struct {
	Usage   uint64
	Max     uint64
	Failcnt uint64
	Limit   uint64
}
type MemoryStat struct {
	Usage     *MemoryEntry
	Swap      *MemoryEntry
	Kernel    *MemoryEntry
	KernelTCP *MemoryEntry

	Cache                   uint64
	RSS                     uint64
	RSSHuge                 uint64
	MappedFile              uint64
	Dirty                   uint64
	Writeback               uint64
	PgPgIn                  uint64
	PgPgOut                 uint64
	PgFault                 uint64
	PgMajFault              uint64
	InactiveAnon            uint64
	ActiveAnon              uint64
	InactiveFile            uint64
	ActiveFile              uint64
	Unevictable             uint64
	HierarchicalMemoryLimit uint64
	HierarchicalSwapLimit   uint64
	TotalCache              uint64
	TotalRSS                uint64
	TotalRSSHuge            uint64
	TotalMappedFile         uint64
	TotalDirty              uint64
	TotalWriteback          uint64
	TotalPgPgIn             uint64
	TotalPgPgOut            uint64
	TotalPgFault            uint64
	TotalPgMajFault         uint64
	TotalInactiveAnon       uint64
	TotalActiveAnon         uint64
	TotalInactiveFile       uint64
	TotalActiveFile         uint64
	TotalUnevictable        uint64
}
type NetworkStat struct {
	Name      string
	RxBytes   uint64
	RxPackets uint64
	RxErrors  uint64
	RxDropped uint64
	TxBytes   uint64
	TxPackets uint64
	TxErrors  uint64
	TxDropped uint64
}
type PidsStat struct {
	Current uint64
	Limit   uint64
}
type Stats struct {
	Blkio  *BlkIOStat
	CPU    *CPUStat
	Memory *MemoryStat
	Pids   *PidsStat
}
