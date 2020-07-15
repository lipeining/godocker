/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package cgroups

// Name is a typed name for a cgroup subsystem
type Name string

const (
	Devices   Name = "devices"
	Hugetlb   Name = "hugetlb"
	Freezer   Name = "freezer"
	Pids      Name = "pids"
	NetCLS    Name = "net_cls"
	NetPrio   Name = "net_prio"
	PerfEvent Name = "perf_event"
	Cpuset    Name = "cpuset"
	Cpu       Name = "cpu"
	Cpuacct   Name = "cpuacct"
	Memory    Name = "memory"
	Blkio     Name = "blkio"
	Rdma      Name = "rdma"
)

// Subsystems returns a complete list of the default cgroups
// available on most linux systems
func Subsystems() []Name {
	n := []Name{
		Pids,
		NetCLS,
		NetPrio,
		Cpuset,
		Cpu,
		Cpuacct,
		Memory,
		Blkio,
	}
	return n
}

type Subsystem interface {
	Name() Name
}

type pather interface {
	Subsystem
	Path(path string) string
}

type creator interface {
	Subsystem
	Create(path string, resources *Resources) error
}

type deleter interface {
	Subsystem
	Delete(path string) error
}

type stater interface {
	Subsystem
	Stat(path string, stats *Stats) error
}

type updater interface {
	Subsystem
	Update(path string, resources *Resources) error
}
