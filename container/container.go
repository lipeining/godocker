package container

import (
	"os"
	"sync"
	"time"

	"github.com/lipeining/godocker/configs"
)

// BaseContainer is a libcontainer container object.
//
// Each container is thread-safe within the same process. Since a container can
// be destroyed by a separate process, any function may return that the container
// was not found. BaseContainer includes methods that are platform agnostic.
type BaseContainer interface {
	// Returns the ID of the container
	ID() string

	// // Returns the current status of the container.
	// //
	// // errors:
	// // ContainerNotExists - Container no longer exists,
	// // Systemerror - System error.
	// Status() (Status, error)

	// // State returns the current container's state information.
	// //
	// // errors:
	// // SystemError - System error.
	// State() (*State, error)

	// // OCIState returns the current container's state information.
	// //
	// // errors:
	// // SystemError - System error.
	// OCIState() (*specs.State, error)

	// // Returns the current config of the container.
	// Config() configs.Config

	// Returns the PIDs inside this container. The PIDs are in the namespace of the calling process.
	//
	// errors:
	// ContainerNotExists - Container no longer exists,
	// Systemerror - System error.
	//
	// Some of the returned PIDs may no longer refer to processes in the Container, unless
	// the Container state is PAUSED in which case every PID in the slice is valid.
	Processes() ([]int, error)

	// // Returns statistics for the container.
	// //
	// // errors:
	// // ContainerNotExists - Container no longer exists,
	// // Systemerror - System error.
	// Stats() (*Stats, error)

	// // Set resources of container as configured
	// //
	// // We can use this to change resources when containers are running.
	// //
	// // errors:
	// // SystemError - System error.
	// Set(config configs.Config) error

	// Start a process inside the container. Returns error if process fails to
	// start. You can track process lifecycle with passed Process structure.
	//
	// errors:
	// ContainerNotExists - Container no longer exists,
	// ConfigInvalid - config is invalid,
	// ContainerPaused - Container is paused,
	// SystemError - System error.
	Start(process *Process) (err error)

	// Run immediately starts the process inside the container.  Returns error if process
	// fails to start.  It does not block waiting for the exec fifo  after start returns but
	// opens the fifo after start returns.
	//
	// errors:
	// ContainerNotExists - Container no longer exists,
	// ConfigInvalid - config is invalid,
	// ContainerPaused - Container is paused,
	// SystemError - System error.
	Run(process *Process) (err error)

	// Destroys the container, if its in a valid state, after killing any
	// remaining running processes.
	//
	// Any event registrations are removed before the container is destroyed.
	// No error is returned if the container is already destroyed.
	//
	// Running containers must first be stopped using Signal(..).
	// Paused containers must first be resumed using Resume(..).
	//
	// errors:
	// ContainerNotStopped - Container is still running,
	// ContainerPaused - Container is paused,
	// SystemError - System error.
	Destroy() error

	// Signal sends the provided signal code to the container's initial process.
	//
	// If all is specified the signal is sent to all processes in the container
	// including the initial process.
	//
	// errors:
	// SystemError - System error.
	Signal(s os.Signal, all bool) error

	// Exec signals the container to exec the users process at the end of the init.
	//
	// errors:
	// SystemError - System error.
	Exec() error
}

// Container is a libcontainer container object.
//
// Each container is thread-safe within the same process. Since a container can
// be destroyed by a separate process, any function may return that the container
// was not found.
type Container interface {
	BaseContainer

	// // Methods below here are platform specific

	// // Checkpoint checkpoints the running container's state to disk using the criu(8) utility.
	// //
	// // errors:
	// // Systemerror - System error.
	// Checkpoint(criuOpts *CriuOpts) error

	// // Restore restores the checkpointed container to a running state using the criu(8) utility.
	// //
	// // errors:
	// // Systemerror - System error.
	// Restore(process *Process, criuOpts *CriuOpts) error

	// If the Container state is RUNNING or CREATED, sets the Container state to PAUSING and pauses
	// the execution of any user processes. Asynchronously, when the container finished being paused the
	// state is changed to PAUSED.
	// If the Container state is PAUSED, do nothing.
	//
	// errors:
	// ContainerNotExists - Container no longer exists,
	// ContainerNotRunning - Container not running or created,
	// Systemerror - System error.
	Pause() error

	// If the Container state is PAUSED, resumes the execution of any user processes in the
	// Container before setting the Container state to RUNNING.
	// If the Container state is RUNNING, do nothing.
	//
	// errors:
	// ContainerNotExists - Container no longer exists,
	// ContainerNotPaused - Container is not paused,
	// Systemerror - System error.
	Resume() error

	// NotifyOOM returns a read-only channel signaling when the container receives an OOM notification.
	//
	// errors:
	// Systemerror - System error.
	NotifyOOM() (<-chan struct{}, error)

	// // NotifyMemoryPressure returns a read-only channel signaling when the container reaches a given pressure level
	// //
	// // errors:
	// // Systemerror - System error.
	// NotifyMemoryPressure(level PressureLevel) (<-chan struct{}, error)
}

type linuxContainer struct {
	id     string
	root   string
	config *configs.Config
	// cgroupManager        cgroups.Manager
	// intelRdtManager      intelrdt.Manager
	initPath string
	initArgs []string
	// initProcess          parentProcess
	initProcessStartTime uint64
	criuPath             string
	newuidmapPath        string
	newgidmapPath        string
	m                    sync.Mutex
	criuVersion          int
	// state                containerState
	created time.Time
}

func startContainer() {

}

func createContainer() {

}
