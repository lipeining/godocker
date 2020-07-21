package container

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lipeining/godocker/cgroups"
	"github.com/lipeining/godocker/configs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"golang.org/x/sys/unix"
)

const (
	StateFilename = "state.json"
)

// Status is the status of a container.
type Status int

const (
	// Created is the status that denotes the container exists but has not been run yet.
	Created Status = iota
	// Running is the status that denotes the container exists and is running.
	Running
	// Pausing is the status that denotes the container exists, it is in the process of being paused.
	Pausing
	// Paused is the status that denotes the container exists, but all its processes are paused.
	Paused
	// Stopped is the status that denotes the container does not have a created or running process.
	Stopped
)

func (s Status) String() string {
	switch s {
	case Created:
		return "created"
	case Running:
		return "running"
	case Pausing:
		return "pausing"
	case Paused:
		return "paused"
	case Stopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// BaseState represents the platform agnostic pieces relating to a
// running container's state
type BaseState struct {
	// ID is the container ID.
	ID string `json:"id"`

	// InitProcessPid is the init process id in the parent namespace.
	InitProcessPid int `json:"init_process_pid"`

	// InitProcessStartTime is the init process start time in clock cycles since boot time.
	InitProcessStartTime uint64 `json:"init_process_start"`

	// Created is the unix timestamp for the creation time of the container in UTC
	Created time.Time `json:"created"`

	// Config is the container's configuration.
	Config configs.Config `json:"config"`
}

// State represents a running container's state
type State struct {
	BaseState

	// Platform specific fields below here

	// Specified if the container was started under the rootless mode.
	// Set to true if BaseState.Config.RootlessEUID && BaseState.Config.RootlessCgroups
	Rootless bool `json:"rootless"`

	// Paths to all the container's cgroups, as returned by (*cgroups.Manager).GetPaths
	//
	// For cgroup v1, a key is cgroup subsystem name, and the value is the path
	// to the cgroup for this subsystem.
	//
	// For cgroup v2 unified hierarchy, a key is "", and the value is the unified path.
	CgroupPaths map[string]string `json:"cgroup_paths"`

	// NamespacePaths are filepaths to the container's namespaces. Key is the namespace type
	// with the value as the path.
	NamespacePaths map[configs.NamespaceType]string `json:"namespace_paths"`
}

type Container struct {
	Id            string
	Name          string
	Root          string
	Config        *configs.Config
	cgroupManager cgroups.Manager
	InitPath      string
	InitArgs      []string
	InitProcess   InitProcess
	m             sync.Mutex
	State         State
	Created       time.Time
}

func NewContainer(context *cli.Context, config *configs.Config, cgroupManager cgroups.Manager) (*Container, error) {
	return &Container{
		Id:            context.String("id"),
		Name:          context.String("name"),
		Root:          context.GlobalString("root"),
		Config:        config,
		cgroupManager: cgroupManager,
		InitPath:      "/proc/self/exe",
		InitArgs:      []string{"init"},
		Created:       time.Now(),
	}, nil
}

// initConfig is used for transferring parameters from Exec() to Init()
type initConfig struct {
	Args        []string        `json:"args"`
	Env         []string        `json:"env"`
	Cwd         string          `json:"cwd"`
	Config      *configs.Config `json:"config"`
	ContainerId string          `json:"containerid"`
}

// StartInit start the /proc/self/exe init
func (c *Container) StartInit(process *InitProcess) error {

	if err := process.cmd.Start(); err != nil {
		logrus.Errorf("parent start failed, err: %v", err)
		return err
	}
	// 在 messageSockPair 中写入对应的启动命令
	// 直接写容器 state，让子容器启动之后，进行读取即可
	// 这里是否需要通过 messageSockPair 控制读写文件的先后顺序
	// 通过 execFifo 进行控制

	// cgroupManager apply
	// 保存 state 让容器启动时，可以得到配置，可以发送 writePipe config
	initConfig := initConfig{
		Args:        process.process.Args,
		Env:         process.process.Env,
		Cwd:         process.process.Cwd,
		Config:      c.Config,
		ContainerId: c.Id,
	}
	WriteJSON(process.messageSockPair.child, initConfig)

	if err := process.manager.Apply(process.cmd.Process.Pid); err != nil {
		return err
	}
	// 启动进程，记录 state 到文件中
	c.UpdateState(*process)
	// if err := process.messageSockPair.child.Close(); err != nil {
	// 	return err
	// }
	// network  volunme 在子容器进行
	// 如果是 tty 的话，需要使用 Wait() 等待进程结束
	// if err := process.cmd.Wait(); err != nil {
	// 	return err
	// }
	return nil
}

func (c *Container) UpdateState(process InitProcess) (*State, error) {
	if process.cmd != nil {
		c.InitProcess = process
	}
	state, err := c.CurrentState()
	if err != nil {
		return nil, err
	}
	err = c.SaveState(state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (c *Container) SaveState(s *State) (retErr error) {
	tmpFile, err := ioutil.TempFile(c.Root, "state-")
	if err != nil {
		return err
	}

	defer func() {
		if retErr != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}
	}()

	err = WriteJSON(tmpFile, s)
	if err != nil {
		return err
	}
	err = tmpFile.Close()
	if err != nil {
		return err
	}

	stateFilePath := filepath.Join(c.Root, StateFilename)
	return os.Rename(tmpFile.Name(), stateFilePath)
}

func (c *Container) DeleteState() error {
	return os.Remove(filepath.Join(c.Root, StateFilename))
}

func (c *Container) CurrentState() (*State, error) {
	var (
		pid = -1
	)
	if c.InitProcess.cmd != nil {
		pid = c.InitProcess.cmd.Process.Pid
		// 需要读取 /proc/:pid/stat
	}
	state := &State{
		BaseState: BaseState{
			ID:                   c.ID(),
			Config:               *c.Config,
			InitProcessPid:       pid,
			InitProcessStartTime: uint64(c.Created.Unix()),
			Created:              c.Created,
		},
		Rootless:       c.Config.RootlessEUID && c.Config.RootlessCgroups,
		NamespacePaths: make(map[configs.NamespaceType]string),
	}
	if pid > 0 {
		for _, ns := range c.Config.Namespaces {
			state.NamespacePaths[ns.Type] = ns.GetPath(pid)
		}
		for _, nsType := range configs.NamespaceTypes() {
			if !configs.IsNamespaceSupported(nsType) {
				continue
			}
			if _, ok := state.NamespacePaths[nsType]; !ok {
				ns := configs.Namespace{Type: nsType}
				state.NamespacePaths[ns.Type] = ns.GetPath(pid)
			}
		}
	}
	return state, nil
}

// refreshState needs to be called to verify that the current state on the
// container is what is true.  Because consumers of libcontainer can use it
// out of process we need to verify the container's status based on runtime
// information and not rely on our in process info.
func (c *Container) RefreshState() error {
	return nil
}

func (c *Container) CurrentStatus() (Status, error) {
	return c.runType(), nil
}
func (c *Container) runType() Status {
	if c.InitProcess.cmd == nil {
		return Stopped
	}
	pid := c.InitProcess.cmd.Process.Pid
	stat, err := Stat(pid)
	if err != nil {
		return Stopped
	}
	if stat.ProcState == ProcZombie || stat.ProcState == ProcDead {
		return Stopped
	}
	// // We'll create exec fifo and blocking on it after container is created,
	// // and delete it after start container.
	// if _, err := os.Stat(filepath.Join(c.Root, execFifoFilename)); err == nil {
	// 	return Created
	// }
	return Running
}

func (c *Container) isPaused() (bool, error) {
	return false, nil
}

// WriteJSON writes the provided struct v to w using standard json marshaling
func WriteJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func StartInitialization() (err error) {
	var (
		pipefd      int
		envInitPipe = os.Getenv("_LIBCONTAINER_INITPIPE")
	)

	// Get the INITPIPE.
	pipefd, err = strconv.Atoi(envInitPipe)
	// hack into fd=3
	pipefd = 3
	if err != nil {
		return fmt.Errorf("unable to convert _LIBCONTAINER_INITPIPE=%s to int: %s", envInitPipe, err)
	}
	var (
		pipe = os.NewFile(uintptr(pipefd), "pipe")
	)
	defer pipe.Close()

	// clear the current process's environment to clean any libcontainer
	// specific env vars.
	os.Clearenv()

	defer func() {
		// We have an error during the initialization of the container's init,
		// send it back to the parent process in the form of an initError.
		// if werr := WriteJSON(pipe, err); werr != nil {
		// 	fmt.Fprintln(os.Stderr, err)
		// 	return
		// }
		// if werr := WriteJSON(pipe, err); werr != nil {
		// 	fmt.Fprintln(os.Stderr, err)
		// 	return
		// }
	}()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic from initialization: %v, %v", e, string(debug.Stack()))
		}
	}()
	return Initialization(pipe)
}
func setupNetwork(config *initConfig) error {
	return nil
}
func setupRoute(config *initConfig) error {
	return nil
}
func prepareRootfs(pipe *os.File, config *initConfig) error {
	return nil
}
func finalizeRootfs(config *initConfig) error {
	return nil
}

// writeSystemProperty writes the value to a path under /proc/sys as determined from the key.
// For e.g. net.ipv4.ip_forward translated to /proc/sys/net/ipv4/ip_forward.
func writeSystemProperty(key, value string) error {
	keyPath := strings.Replace(key, ".", "/", -1)
	return ioutil.WriteFile(path.Join("/proc/sys", keyPath), []byte(value), 0644)
}

func Initialization(pipe *os.File) (err error) {
	// 对应的格式进行 init
	var config *initConfig
	if err := json.NewDecoder(pipe).Decode(&config); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := setupNetwork(config); err != nil {
		return err
	}
	if err := setupRoute(config); err != nil {
		return err
	}
	if err := prepareRootfs(pipe, config); err != nil {
		return err
	}
	if err := setUpMount(config); err != nil {
		return err
	}
	// Finish the rootfs setup.
	if config.Config.Namespaces.Contains(configs.NEWNS) {
		if err := finalizeRootfs(config); err != nil {
			return err
		}
	}
	if hostname := config.Config.Hostname; hostname != "" {
		if err := unix.Sethostname([]byte(hostname)); err != nil {
			return errors.Wrap(err, "sethostname")
		}
	}
	for key, value := range config.Config.Sysctl {
		if err := writeSystemProperty(key, value); err != nil {
			return errors.Wrapf(err, "write sysctl key %s", key)
		}
	}
	for _, path := range config.Config.ReadonlyPaths {
		if err := readonlyPath(path); err != nil {
			return errors.Wrapf(err, "readonly path %s", path)
		}
	}
	for _, path := range config.Config.MaskPaths {
		if err := maskPath(path, config.Config.MountLabel); err != nil {
			return errors.Wrapf(err, "mask path %s", path)
		}
	}
	// // Close the pipe to signal that we have completed our init.
	// pipe.Close()
	return unix.Exec(config.Args[0], config.Args[0:], os.Environ())
}
func setUpMount(config *initConfig) error {
	err := pivotRoot(config.Config.Rootfs)
	if err != nil {
		logrus.Errorf("pivot root, err: %v", err)
		return err
	}

	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示
	//声明你要这个新的mount namespace独立。
	err = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return err
	}
	//mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Errorf("mount proc, err: %v", err)
		return err
	}
	// mount temfs, temfs是一种基于内存的文件系统
	err = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	if err != nil {
		logrus.Errorf("mount tempfs, err: %v", err)
		return err
	}

	return nil
}

//
func (c *Container) ID() string {
	// 启动进程，记录 state 到文件中
	return c.Id
}
func (c *Container) Start(process *Process) error {
	// 启动进程，记录 state 到文件中
	return nil
}
func (c *Container) Signal(signal os.Signal, all bool) error {
	// 启动进程，记录 state 到文件中
	c.m.Lock()
	defer c.m.Unlock()
	status, err := c.CurrentStatus()
	if err != nil {
		return err
	}
	if all {
		// for systemd cgroup, the unit's cgroup path will be auto removed if container's all processes exited
		if status == Stopped {
			return nil
		}
		// if status == Stopped && !c.cgroupManager.Exists() {
		// 	return nil
		// }
		return nil
	}
	// to avoid a PID reuse attack
	if status == Running || status == Created || status == Paused {
		if err := c.InitProcess.cmd.Process.Signal(signal); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (c *Container) Pause() error {
	return nil
}

func (c *Container) Resume() error {
	return nil
}
