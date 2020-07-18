// +build linux

package container

import (
	"fmt"
	"os"
	"runtime"
)

// linuxSetnsInit performs the container's initialization for running a new process
// inside an existing container.
type linuxSetnsInit struct {
	pipe          *os.File
}

func (l *linuxSetnsInit) getSessionRingName() string {
	return fmt.Sprintf("_ses.%s", "1")
}

func (l *linuxSetnsInit) Init() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return nil
	// return system.Execv(l.config.Args[0], l.config.Args[0:], os.Environ())
}
