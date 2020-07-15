package cgroups

type BlockIOResource struct {
	Weight     uint64
	LeafWeight uint64
}
type CpuResource struct {
	RealtimePeriod  *uint64
	RealtimeRuntime *int64
	Shares          *uint64
	Period          *uint64
	Quota           *int64
	Cpus            string
	Mems            string
}
type MemoryResource struct {
	Kernel           *int64
	KernelTCP        *int64
	Limit            *int64
	Swap             *int64
	Swappiness       *int64
	Reservation      *int64
	DisableOOMKiller *bool
}

type NetclsResource struct {
	ClassID    *uint32
	Priorities []InterfacePriority
}
type InterfacePriority struct {
	Name     string
	Priority uint32
}
type PidsResource struct {
	Limit int64
}
type Resources struct {
	BlockIO *BlockIOResource
	CPU     *CpuResource
	Memory  *MemoryResource
	Network *NetclsResource
	Pids    *PidsResource
}
