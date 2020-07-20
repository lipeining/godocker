package container

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lipeining/godocker/cgroups"
	"github.com/lipeining/godocker/configs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"golang.org/x/sys/unix"
)

const (
	stateFilename = "state.json"
)

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
		Root:          context.String("root"),
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
	// 启动进程，记录 state 到文件中
	c.InitProcess = *process
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
	// if err := process.messageSockPair.child.Close(); err != nil {
	// 	return err
	// }
	// network  volunme 在子容器进行
	// if err := process.cmd.Wait(); err != nil {
	// 	return err
	// }
	return nil
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
		// if werr := WriteJSON(pipe, 1); werr != nil {
		// 	fmt.Fprintln(os.Stderr, err)
		// 	return
		// }
		// if werr := WriteJSON(pipe, 1); werr != nil {
		// 	fmt.Fprintln(os.Stderr, err)
		// 	return
		// }
	}()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic from initialization: %v, %v", e, string(debug.Stack()))
		}
	}()

	return nil
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
	// Close the pipe to signal that we have completed our init.
	pipe.Close()
	return unix.Exec(config.Args[0], config.Args[0:], os.Environ())
}

//
func (c *Container) Start(process *Process) error {
	// 启动进程，记录 state 到文件中
	return nil
}

func (c *Container) loadState() error {
	return nil
}
func (c *Container) updateState() error {
	return nil
}
func (c *Container) saveState() error {
	return nil
}
