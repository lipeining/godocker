package main

import (
	"os"
	"runtime"

	"github.com/lipeining/godocker/container"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		logrus.Debug("child process in init()")
	}
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: `initialize the namespaces and launch the process (do not call it outside of runc)`,
	Action: func(context *cli.Context) error {
		logrus.Info("godocker come to init command")
		// parent process call /proc/self/exe init
		// we should use env to pass parent child pipe
		// or just the pipe3 to pass config from parent to child
		// now just start the user cmd
		err := container.StartInitialization()
		return err
	},
}
