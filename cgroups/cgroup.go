package cgroups

import (
	"os"
	"sync"
)

// Cgroup handles interactions with the individual groups to perform
// actions on them as them main interface to this cgroup package
type Cgroup interface {
	// // New creates a new cgroup under the calling cgroup
	// New(string, *Resources) (Cgroup, error)
	// // Add adds a process to the cgroup (cgroup.procs)
	// Add(Process) error
	// // AddTask adds a process to the cgroup (tasks)
	// AddTask(Process) error
	// // Delete removes the cgroup as a whole
	// Delete() error
	// // MoveTo moves all the processes under the calling cgroup to the provided one
	// // subsystems are moved one at a time
	// MoveTo(Cgroup) error
	// // Stat returns the stats for all subsystems in the cgroup
	// Stat(...ErrorHandler) (*Stats, error)
}

// cgroup hold a cgroup manager
type cgroup struct {
	path       string
	subsystems []Subsystem
	mu         sync.Mutex
	err        error
}

// NewCgroup return a cgroup
func NewCgroup(path string, resources *Resources) (Cgroup, error) {
	root, err := getMountPoint()
	if err != nil {
		return nil, err
	}
	subsystems, err := defaults(root)
	if err != nil {
		return nil, err
	}
	var enabled []Subsystem
	for _, s := range pathers(subsystems) {
		// check and remove the default groups that do not exist
		if _, err := os.Lstat(s.Path("/")); err == nil {
			enabled = append(enabled, s)
		}
	}
	var active []Subsystem
	for _, s := range enabled {
		// check if subsystem exists
		if err := initializeSubsystem(s, path, resources); err != nil {
			return nil, err
		}
		active = append(active, s)
	}
	return &cgroup{
		path:       path,
		subsystems: active,
	}, nil
}
