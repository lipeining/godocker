package container

import (
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"

	securejoin "github.com/cyphar/filepath-securejoin"
	"golang.org/x/sys/unix"
)

const (
	stateFilename = "state.json"
)

var idRegex = regexp.MustCompile(`^[\w+-\.]+$`)

type Factory interface {
	// Creates a new container with the given id and starts the initial process inside it.
	// id must be a string containing only letters, digits and underscores and must contain
	// between 1 and 1024 characters, inclusive.
	//
	// The id must not already be in use by an existing container. Containers created using
	// a factory with the same path (and filesystem) must have distinct ids.
	//
	// Returns the new container with a running process.
	//
	// errors:
	// IdInUse - id is already in use by a container
	// InvalidIdFormat - id has incorrect format
	// ConfigInvalid - config is invalid
	// Systemerror - System error
	//
	// On error, any partially created container parts are cleaned up (the operation is atomic).
	Create(id string) (Container, error)

	// Load takes an ID for an existing container and returns the container information
	// from the state.  This presents a read only view of the container.
	//
	// errors:
	// Path does not exist
	// System error
	Load(id string) (Container, error)

	// StartInitialization is an internal API to libcontainer used during the reexec of the
	// container.
	//
	// Errors:
	// Pipe connection error
	// System error
	// StartInitialization() error

	// Type returns info string about factory type (e.g. lxc, libcontainer...)
	// Type() string
}

// New returns a linux based container factory based in the root directory and
// configures the factory with the provided option funcs.
func New(root string, options ...func(*LinuxFactory) error) (Factory, error) {
	if root != "" {
		if err := os.MkdirAll(root, 0700); err != nil {
			return nil, err
		}
	}
	l := &LinuxFactory{
		Root:     root,
		InitPath: "/proc/self/exe",
		InitArgs: []string{os.Args[0], "init"},
	}
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if err := opt(l); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// LinuxFactory implements the default factory interface for linux based systems.
type LinuxFactory struct {
	// Root directory for the factory to store state.
	Root string

	// InitPath is the path for calling the init responsibilities for spawning
	// a container.
	InitPath string

	// InitArgs are arguments for calling the init responsibilities for spawning
	// a container.
	InitArgs []string
}

func (l *LinuxFactory) Create(id string) (Container, error) {
	if l.Root == "" {
		return nil, nil
	}
	if err := l.validateID(id); err != nil {
		return nil, err
	}
	containerRoot, err := securejoin.SecureJoin(l.Root, id)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(containerRoot); err != nil {
		return nil, err
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(containerRoot, 0711); err != nil {
		return nil, err
	}
	if err := os.Chown(containerRoot, unix.Geteuid(), unix.Getegid()); err != nil {
		return nil, err
	}
	c := &linuxContainer{
		id:       id,
		root:     containerRoot,
		initPath: l.InitPath,
		initArgs: l.InitArgs,
	}
	return c, nil
}

func (l *LinuxFactory) Load(id string) (Container, error) {
	if l.Root == "" {
		return nil, nil
	}
	if err := l.validateID(id); err != nil {
		return nil, err
	}
	containerRoot, err := securejoin.SecureJoin(l.Root, id)
	if err != nil {
		return nil, err
	}
	// state, err := l.loadState(containerRoot, id)
	// if err != nil {
	// 	return nil, err
	// }
	// r := &Process{
	// 	processPid:       state.InitProcessPid,
	// 	processStartTime: state.InitProcessStartTime,
	// 	fds:              state.ExternalDescriptors,
	// }
	c := &linuxContainer{
		// initProcess:          r,
		// initProcessStartTime: state.InitProcessStartTime,
		id: id,
		// config:               &state.Config,
		initPath: l.InitPath,
		initArgs: l.InitArgs,
		root:     containerRoot,
	}
	// c.state = &loadedState{c: c}
	// if err := c.refreshState(); err != nil {
	// 	return nil, err
	// }
	// if intelrdt.IsCatEnabled() || intelrdt.IsMbaEnabled() {
	// 	c.intelRdtManager = l.NewIntelRdtManager(&state.Config, id, state.IntelRdtPath)
	// }
	return c, nil
}

// StartInitialization loads a container by opening the pipe fd from the parent to read the configuration and state
// This is a low level implementation detail of the reexec and should not be consumed externally
func (l *LinuxFactory) StartInitialization() (err error) {
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

	// Only init processes have FIFOFD.

	// clear the current process's environment to clean any libcontainer
	// specific env vars.
	os.Clearenv()

	defer func() {
		// We have an error during the initialization of the container's init,
		// send it back to the parent process in the form of an initError.
		// if werr := utils.WriteJSON(pipe, errors.New("system")); werr != nil {
		// 	fmt.Fprintln(os.Stderr, err)
		// 	return
		// }
		// if werr := utils.WriteJSON(pipe, err); werr != nil {
		// 	fmt.Fprintln(os.Stderr, err)
		// 	return
		// }
	}()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic from initialization: %v, %v", e, string(debug.Stack()))
		}
	}()

	// i, err := newContainerInit(it, pipe, consoleSocket, fifofd)
	// if err != nil {
	// 	return err
	// }

	// If Init succeeds, syscall.Exec will not return, hence none of the defers will be called.
	// return i.Init()
	return nil
}

func (l *LinuxFactory) loadState(root, id string) (error, error) {
	stateFilePath, err := securejoin.SecureJoin(root, stateFilename)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, err
	}
	defer f.Close()
	return nil, nil
	// var state *State
	// if err := json.NewDecoder(f).Decode(&state); err != nil {
	// 	return nil, newGenericError(err, SystemError)
	// }
	// return state, nil
}

func (l *LinuxFactory) validateID(id string) error {
	if !idRegex.MatchString(id) {
		return fmt.Errorf("invalid id format: %v", id)
	}

	return nil
}
