package container

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/lipeining/godocker/cgroups"
	"github.com/sirupsen/logrus"
)

// Process specifies the configuration and IO for a process inside
// a container.
type Process struct {
	// The command to be run followed by any arguments.
	Args []string

	// Env specifies the environment variables for the process.
	Env []string

	// Cwd will change the processes current working directory inside the container's rootfs.
	Cwd string
}

type filePair struct {
	parent *os.File
	child  *os.File
}

type InitProcess struct {
	cmd             *exec.Cmd
	manager         cgroups.Manager
	messageSockPair filePair
	process         *Process
}

type SetnsProcess struct {
}

const (
	defaultContainerInfoPath    = "/run/godocker/"
	defaultContainerLogFileName = "log"
)

func NewProcess(args, env []string, cwd string) *Process {
	return &Process{
		Cwd:  cwd,
		Args: args,
		Env:  env,
	}
}

// NewInitProcess create a process to init
func NewInitProcess(process *Process, containerName string) (*InitProcess, error) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	// 调用自身，传入 init 参数，也就是执行 initCommand
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	tty := true
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// 把日志输出到文件里
		logDir := filepath.Join(defaultContainerInfoPath, containerName)
		if _, err := os.Stat(logDir); err != nil && os.IsNotExist(err) {
			err := os.MkdirAll(logDir, os.ModePerm)
			if err != nil {
				logrus.Errorf("mkdir container log, err: %v", err)
				return nil, err
			}
		}
		logFileName := filepath.Join(logDir, defaultContainerLogFileName)
		file, err := os.Create(logFileName)
		if err != nil {
			logrus.Errorf("create log file, err: %v", err)
			return nil, err
		}
		cmd.Stdout = file
	}
	// set parent child pipe which use to pass config
	// how about fix process.ExtraFiles
	cmd.ExtraFiles = []*os.File{
		readPipe,
	}
	cmd.Env = append(os.Environ(), process.Env...)
	cmd.Dir = process.Cwd
	return &InitProcess{
		cmd:             cmd,
		messageSockPair: filePair{readPipe, writePipe},
		process:         process,
	}, nil
}
