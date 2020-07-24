package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Cgroup hold a cgroup manager
type Cgroup struct {
	path       string
	subsystems []Subsystem
	mu         sync.Mutex
	err        error
}

// NewCgroup return a cgroup
func NewCgroup(path string, resources *Resources) (*Cgroup, error) {
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
	return &Cgroup{
		path:       path,
		subsystems: active,
	}, nil
}

// Subsystems returns all the subsystems that are currently being
// consumed by the group
func (c *Cgroup) Subsystems() []Subsystem {
	return c.subsystems
}

func (c *Cgroup) Path(name Name) (string, error) {
	return c.path, nil
}

// Add moves the provided process into the new cgroup
func (c *Cgroup) Add(process Process) error {
	if process.Pid <= 0 {
		return ErrInvalidPid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	return c.add(process)
}

func (c *Cgroup) add(process Process) error {
	for _, s := range pathers(c.subsystems) {
		p, err := c.Path(s.Name())
		if err != nil {
			return err
		}
		if err := retryingWriteFile(
			filepath.Join(s.Path(p), cgroupProcs),
			[]byte(strconv.Itoa(process.Pid)),
			defaultFilePerm,
		); err != nil {
			return err
		}
	}
	return nil
}

// AddTask moves the provided tasks (threads) into the new cgroup
func (c *Cgroup) AddTask(process Process) error {
	if process.Pid <= 0 {
		return ErrInvalidPid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	return c.addTask(process)
}

func (c *Cgroup) addTask(process Process) error {
	for _, s := range pathers(c.subsystems) {
		p, err := c.Path(s.Name())
		if err != nil {
			return err
		}
		if err := retryingWriteFile(
			filepath.Join(s.Path(p), cgroupTasks),
			[]byte(strconv.Itoa(process.Pid)),
			defaultFilePerm,
		); err != nil {
			return err
		}
	}
	return nil
}

// Delete will remove the control group from each of the subsystems registered
func (c *Cgroup) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	var errs []string
	for _, s := range c.subsystems {
		if d, ok := s.(deleter); ok {
			sp, err := c.Path(s.Name())
			if err != nil {
				return err
			}
			if err := d.Delete(sp); err != nil {
				errs = append(errs, string(s.Name()))
			}
			continue
		}
		if p, ok := s.(pather); ok {
			sp, err := c.Path(s.Name())
			if err != nil {
				return err
			}
			path := p.Path(sp)
			if err := remove(path); err != nil {
				errs = append(errs, path)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("cgroups: unable to remove paths %s", strings.Join(errs, ", "))
	}
	c.err = ErrCgroupDeleted
	return nil
}

// Stat returns the current metrics for the cgroup
func (c *Cgroup) Stat(handlers ...ErrorHandler) (*Stats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	if len(handlers) == 0 {
		handlers = append(handlers, errPassthrough)
	}
	var (
		stats = &Stats{
			CPU: &CPUStat{
				Throttling: &ThrottlingStat{},
				Usage:      &CPUUsage{},
			},
		}
		wg   = &sync.WaitGroup{}
		errs = make(chan error, len(c.subsystems))
	)
	for _, s := range c.subsystems {
		if ss, ok := s.(stater); ok {
			sp, err := c.Path(s.Name())
			if err != nil {
				return nil, err
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := ss.Stat(sp, stats); err != nil {
					for _, eh := range handlers {
						if herr := eh(err); herr != nil {
							errs <- herr
						}
					}
				}
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return nil, err
	}
	return stats, nil
}

// Update updates the cgroup with the new resource values provided
//
// Be prepared to handle EBUSY when trying to update a cgroup with
// live processes and other operations like Stats being performed at the
// same time
func (c *Cgroup) Update(resources *Resources) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	for _, s := range c.subsystems {
		if u, ok := s.(updater); ok {
			sp, err := c.Path(s.Name())
			if err != nil {
				return err
			}
			if err := u.Update(sp, resources); err != nil {
				return err
			}
		}
	}
	return nil
}

// Processes returns the processes running inside the cgroup along
// with the subsystem used, pid, and path
func (c *Cgroup) Processes(subsystem Name, recursive bool) ([]Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	return c.processes(subsystem, recursive)
}

func (c *Cgroup) processes(subsystem Name, recursive bool) ([]Process, error) {
	s := c.getSubsystem(subsystem)
	sp, err := c.Path(subsystem)
	if err != nil {
		return nil, err
	}
	path := s.(pather).Path(sp)
	var processes []Process
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !recursive && info.IsDir() {
			if p == path {
				return nil
			}
			return filepath.SkipDir
		}
		dir, name := filepath.Split(p)
		if name != cgroupProcs {
			return nil
		}
		procs, err := readPids(dir, subsystem)
		if err != nil {
			return err
		}
		processes = append(processes, procs...)
		return nil
	})
	return processes, err
}

// Tasks returns the tasks running inside the cgroup along
// with the subsystem used, pid, and path
func (c *Cgroup) Tasks(subsystem Name, recursive bool) ([]Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return nil, c.err
	}
	return c.tasks(subsystem, recursive)
}

func (c *Cgroup) tasks(subsystem Name, recursive bool) ([]Task, error) {
	s := c.getSubsystem(subsystem)
	sp, err := c.Path(subsystem)
	if err != nil {
		return nil, err
	}
	path := s.(pather).Path(sp)
	var tasks []Task
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !recursive && info.IsDir() {
			if p == path {
				return nil
			}
			return filepath.SkipDir
		}
		dir, name := filepath.Split(p)
		if name != cgroupTasks {
			return nil
		}
		procs, err := readTasksPids(dir, subsystem)
		if err != nil {
			return err
		}
		tasks = append(tasks, procs...)
		return nil
	})
	return tasks, err
}

// // Freeze freezes the entire cgroup and all the processes inside it
// func (c *Cgroup) Freeze() error {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	if c.err != nil {
// 		return c.err
// 	}
// 	s := c.getSubsystem(Freezer)
// 	if s == nil {
// 		return ErrFreezerNotSupported
// 	}
// 	sp, err := c.Path(Freezer)
// 	if err != nil {
// 		return err
// 	}
// 	return s.(*freezerController).Freeze(sp)
// }

// // Thaw thaws out the cgroup and all the processes inside it
// func (c *Cgroup) Thaw() error {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	if c.err != nil {
// 		return c.err
// 	}
// 	s := c.getSubsystem(Freezer)
// 	if s == nil {
// 		return ErrFreezerNotSupported
// 	}
// 	sp, err := c.Path(Freezer)
// 	if err != nil {
// 		return err
// 	}
// 	return s.(*freezerController).Thaw(sp)
// }

// // OOMEventFD returns the memory cgroup's out of memory event fd that triggers
// // when processes inside the cgroup receive an oom event. Returns
// // ErrMemoryNotSupported if memory cgroups is not supported.
// func (c *Cgroup) OOMEventFD() (uintptr, error) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	if c.err != nil {
// 		return 0, c.err
// 	}
// 	s := c.getSubsystem(Memory)
// 	if s == nil {
// 		return 0, ErrMemoryNotSupported
// 	}
// 	sp, err := c.Path(Memory)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return s.(*memoryController).memoryEvent(sp, OOMEvent())
// }

// // RegisterMemoryEvent allows the ability to register for all v1 memory cgroups
// // notifications.
// func (c *Cgroup) RegisterMemoryEvent(event MemoryEvent) (uintptr, error) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	if c.err != nil {
// 		return 0, c.err
// 	}
// 	s := c.getSubsystem(Memory)
// 	if s == nil {
// 		return 0, ErrMemoryNotSupported
// 	}
// 	sp, err := c.Path(Memory)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return s.(*memoryController).memoryEvent(sp, event)
// }

// // State returns the state of the cgroup and its processes
// func (c *Cgroup) State() State {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	c.checkExists()
// 	if c.err != nil && c.err == ErrCgroupDeleted {
// 		return Deleted
// 	}
// 	s := c.getSubsystem(Freezer)
// 	if s == nil {
// 		return Thawed
// 	}
// 	sp, err := c.Path(Freezer)
// 	if err != nil {
// 		return Unknown
// 	}
// 	state, err := s.(*freezerController).state(sp)
// 	if err != nil {
// 		return Unknown
// 	}
// 	return state
// }

// // MoveTo does a recursive move subsystem by subsystem of all the processes
// // inside the group
// func (c *Cgroup) MoveTo(destination Cgroup) error {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	if c.err != nil {
// 		return c.err
// 	}
// 	for _, s := range c.subsystems {
// 		processes, err := c.processes(s.Name(), true)
// 		if err != nil {
// 			return err
// 		}
// 		for _, p := range processes {
// 			if err := destination.Add(p); err != nil {
// 				if strings.Contains(err.Error(), "no such process") {
// 					continue
// 				}
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

func (c *Cgroup) getSubsystem(n Name) Subsystem {
	for _, s := range c.subsystems {
		if s.Name() == n {
			return s
		}
	}
	return nil
}

func (c *Cgroup) checkExists() {
	for _, s := range pathers(c.subsystems) {
		p, err := c.Path(s.Name())
		if err != nil {
			return
		}
		if _, err := os.Lstat(s.Path(p)); err != nil {
			if os.IsNotExist(err) {
				c.err = ErrCgroupDeleted
				return
			}
		}
	}
}
