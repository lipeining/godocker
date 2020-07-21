// +build linux

package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "execute new process inside the container",
	ArgsUsage: `<container-id> <command> [command options]  || -p process.json <container-id>

Where "<container-id>" is the name for the instance of the container and
"<command>" is the command to be executed in the container.
"<command>" can't be empty unless a "-p" flag provided.

EXAMPLE:
For example, if the container is configured to run the linux ps command the
following will output a list of processes running in the container:

       # runc exec <container-id> ps`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "console-socket",
			Usage: "path to an AF_UNIX socket which will receive a file descriptor referencing the master end of the console's pseudoterminal",
		},
		cli.StringFlag{
			Name:  "cwd",
			Usage: "current working directory in the container",
		},
		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "set environment variables",
		},
		cli.BoolFlag{
			Name:  "tty, t",
			Usage: "allocate a pseudo-TTY",
		},
		cli.StringFlag{
			Name:  "user, u",
			Usage: "UID (format: <uid>[:<gid>])",
		},
		cli.Int64SliceFlag{
			Name:  "additional-gids, g",
			Usage: "additional gids",
		},
		cli.StringFlag{
			Name:  "process, p",
			Usage: "path to the process.json",
		},
		cli.BoolFlag{
			Name:  "detach,d",
			Usage: "detach from the container's process",
		},
		cli.StringFlag{
			Name:  "pid-file",
			Value: "",
			Usage: "specify the file to write the process id to",
		},
		cli.StringFlag{
			Name:  "process-label",
			Usage: "set the asm process label for the process commonly used with selinux",
		},
		cli.StringFlag{
			Name:  "apparmor",
			Usage: "set the apparmor profile for the process",
		},
		cli.BoolFlag{
			Name:  "no-new-privs",
			Usage: "set the no new privileges value for the process",
		},
		cli.StringSliceFlag{
			Name:  "cap, c",
			Value: &cli.StringSlice{},
			Usage: "add a capability to the bounding set for the process",
		},
		cli.BoolFlag{
			Name:   "no-subreaper",
			Usage:  "disable the use of the subreaper used to reap reparented processes",
			Hidden: true,
		},
		cli.IntFlag{
			Name:  "preserve-fds",
			Usage: "Pass N additional file descriptors to the container (stdio + $LISTEN_FDS + N in total)",
		},
	},
	Action: func(context *cli.Context) error {
		if err := checkArgs(context, 1, minArgs); err != nil {
			return err
		}
		fmt.Print(context.Args())
		// status, err := execProcess(context)
		// if err == nil {
		// 	os.Exit(status)
		// }
		// return fmt.Errorf("exec failed: %v", err)
		return nil
	},
	SkipArgReorder: true,
}

func execProcess(context *cli.Context) (int, error) {
	return 0, nil
	// container, err := getContainer(context)
	// if err != nil {
	// 	return -1, err
	// }
	// status, err := container.Status()
	// if err != nil {
	// 	return -1, err
	// }
	// if status == libcontainer.Stopped {
	// 	return -1, fmt.Errorf("cannot exec a container that has stopped")
	// }
	// path := context.String("process")
	// if path == "" && len(context.Args()) == 1 {
	// 	return -1, fmt.Errorf("process args cannot be empty")
	// }
	// detach := context.Bool("detach")
	// state, err := container.State()
	// if err != nil {
	// 	return -1, err
	// }
	// bundle := utils.SearchLabels(state.Config.Labels, "bundle")
	// p, err := getProcess(context, bundle)
	// if err != nil {
	// 	return -1, err
	// }

	// logLevel := "info"
	// if context.GlobalBool("debug") {
	// 	logLevel = "debug"
	// }

	// r := &runner{
	// 	enableSubreaper: false,
	// 	shouldDestroy:   false,
	// 	container:       container,
	// 	consoleSocket:   context.String("console-socket"),
	// 	detach:          detach,
	// 	pidFile:         context.String("pid-file"),
	// 	action:          CT_ACT_RUN,
	// 	init:            false,
	// 	preserveFDs:     context.Int("preserve-fds"),
	// 	logLevel:        logLevel,
	// }
	// return r.run(p)
}
