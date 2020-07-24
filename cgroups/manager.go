package cgroups

import "github.com/lipeining/godocker/configs"

type Manager struct {
	Cgroup *Cgroup
	Config *configs.Config
}

func NewResource(config *configs.Config) *Resources {
	if config.Cgroups != nil && config.Cgroups.Resources != nil {
		r := config.Cgroups.Resources
		realtimePeriod := uint64(r.CpuRtPeriod)
		realtimeRuntime := int64(r.CpuRtRuntime)
		period := uint64(r.CpuPeriod)
		shares := uint64(r.CpuShares)
		quota := int64(r.CpuQuota)
		cpuResource := &CpuResource{
			RealtimePeriod:  &realtimePeriod,
			RealtimeRuntime: &realtimeRuntime,
			Shares:          &shares,
			Period:          &period,
			Quota:           &quota,
			Cpus:            r.CpusetCpus,
			Mems:            r.CpusetMems,
		}
		kernel := int64(r.KernelMemory)
		kernelTCP := int64(r.KernelMemoryTCP)
		swappiness := int64(*r.MemorySwappiness)
		swap := int64(r.MemorySwap)
		limit := int64(r.Memory)
		reservation := int64(r.MemoryReservation)
		disableOOMKiller := bool(r.OomKillDisable)
		memoryResource := &MemoryResource{
			Kernel:           &kernel,
			KernelTCP:        &kernelTCP,
			Limit:            &limit,
			Swap:             &swap,
			Swappiness:       &swappiness,
			Reservation:      &reservation,
			DisableOOMKiller: &disableOOMKiller,
		}
		classID := uint32(r.NetClsClassid)
		netclsResource := &NetclsResource{
			ClassID: &classID,
		}
		resources := &Resources{
			CPU:     cpuResource,
			Memory:  memoryResource,
			Network: netclsResource,
		}
		return resources
	}
	return nil
}
func NewManager(path string, config *configs.Config) (*Manager, error) {
	resources := NewResource(config)
	cgroup, err := NewCgroup(path, resources)
	if err != nil {
		return nil, err
	}
	return &Manager{
		Cgroup: cgroup,
		Config: config,
	}, nil
}

func (m *Manager) Apply(pid int) error {
	if err := m.Cgroup.Add(Process{Pid: pid}); err != nil {
		return err
	}
	if err := m.Cgroup.AddTask(Process{Pid: pid}); err != nil {
		return err
	}
	return nil
}
