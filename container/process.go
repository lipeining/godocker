package container

import (
	"os"
	"os/exec"
	"syscall"
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

	// Init
	Init bool
}

type filePair struct {
	parent *os.File
	child  *os.File
}

type InitProcess struct {
	cmd             *exec.Cmd
	messageSockPair filePair
	process         *Process
}

// NewInitProcess create a process to init
func NewInitProcess(process *Process) InitProcess {
	readPipe, writePipe, _ := os.Pipe()
	// 调用自身，传入 init 参数，也就是执行 initCommand
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	// DefaultContainerInfoPath, ContainerLogFileName := "", ""
	// containerName := "1"
	tty := true
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// // 把日志输出到文件里
		// logDir := path.Join(DefaultContainerInfoPath, containerName)
		// if _, err := os.Stat(logDir); err != nil && os.IsNotExist(err) {
		// 	err := os.MkdirAll(logDir, os.ModePerm)
		// 	if err != nil {
		// 		logrus.Errorf("mkdir container log, err: %v", err)
		// 	}
		// }
		// logFileName := path.Join(logDir, ContainerLogFileName)
		// file, err := os.Create(logFileName)
		// if err != nil {
		// 	logrus.Errorf("create log file, err: %v", err)
		// }
		// cmd.Stdout = file
	}
	// set parent child pipe which use to pass config
	// how about fix process.ExtraFiles
	cmd.ExtraFiles = []*os.File{
		readPipe,
	}
	cmd.Env = append(os.Environ(), process.Env...)
	cmd.Dir = process.Cwd
	return InitProcess{
		cmd:             cmd,
		messageSockPair: filePair{readPipe, writePipe},
		process:         process,
	}
}
