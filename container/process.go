package container

import (
	"io"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/sirupsen/logrus"
)

type processOperations interface {
	wait() (*os.ProcessState, error)
	signal(sig os.Signal) error
	pid() int
}

// Process specifies the configuration and IO for a process inside
// a container.
type Process struct {
	// The command to be run followed by any arguments.
	Args []string

	// Env specifies the environment variables for the process.
	Env []string

	// Cwd will change the processes current working directory inside the container's rootfs.
	Cwd string

	// Stdin is a pointer to a reader which provides the standard input stream.
	Stdin io.Reader

	// Stdout is a pointer to a writer which receives the standard output stream.
	Stdout io.Writer

	// Stderr is a pointer to a writer which receives the standard error stream.
	Stderr io.Writer

	// ExtraFiles specifies additional open files to be inherited by the container
	ExtraFiles []*os.File

	// Init specifies whether the process is the first process in the container.
	Init bool

	ops processOperations

	LogLevel string
}
type parentProcess interface {
	// pid returns the pid for the running process.
	pid() int

	// start starts the process execution.
	start() error

	// send a SIGKILL to the process and wait for the exit.
	terminate() error

	// wait waits on the process returning the process state.
	wait() (*os.ProcessState, error)

	// startTime returns the process start time.
	startTime() (uint64, error)

	signal(os.Signal) error

	externalDescriptors() []string

	setExternalDescriptors(fds []string)

	forwardChildLogs()
}

type filePair struct {
	parent *os.File
	child  *os.File
}

type setnsProcess struct {
	cmd             *exec.Cmd
	messageSockPair filePair
	logFilePair     filePair
	cgroupPaths     map[string]string
	rootlessCgroups bool
	intelRdtPath    string
	// config          *initConfig
	fds            []string
	process        *Process
	bootstrapData  io.Reader
	initProcessPid int
}
type initProcess struct {
	cmd             *exec.Cmd
	messageSockPair filePair
	logFilePair     filePair
	// config          *initConfig
	// manager         cgroups.Manager
	// intelRdtManager intelrdt.Manager
	container     *linuxContainer
	fds           []string
	process       *Process
	bootstrapData io.Reader
	sharePidns    bool
}

// NewParentProcess create a process to init
func NewParentProcess(process *Process) (*exec.Cmd, *os.File) {
	readPipe, writePipe, _ := os.Pipe()
	// 调用自身，传入 init 参数，也就是执行 initCommand
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
		// 	syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	DefaultContainerInfoPath, ContainerLogFileName := "", ""
	tty := true
	containerName := "1"
	envs := []string{}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// 把日志输出到文件里
		logDir := path.Join(DefaultContainerInfoPath, containerName)
		if _, err := os.Stat(logDir); err != nil && os.IsNotExist(err) {
			err := os.MkdirAll(logDir, os.ModePerm)
			if err != nil {
				logrus.Errorf("mkdir container log, err: %v", err)
			}
		}
		logFileName := path.Join(logDir, ContainerLogFileName)
		file, err := os.Create(logFileName)
		if err != nil {
			logrus.Errorf("create log file, err: %v", err)
		}
		cmd.Stdout = file
	}
	// 设置额外文件句柄
	cmd.ExtraFiles = []*os.File{
		readPipe,
	}
	// 设置环境变量
	cmd.Env = append(os.Environ(), envs...)
	// 指定容器初始化后的工作目录
	// cmd.Dir = common.MntPath
	return cmd, writePipe
}
