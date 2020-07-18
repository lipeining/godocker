package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lipeining/godocker/container"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	exactArgs = iota
	minArgs
	maxArgs
)

func checkArgs(context *cli.Context, expected, checkType int) error {
	var err error
	cmdName := context.Command.Name
	switch checkType {
	case exactArgs:
		if context.NArg() != expected {
			err = fmt.Errorf("%s: %q requires exactly %d argument(s)", os.Args[0], cmdName, expected)
		}
	case minArgs:
		if context.NArg() < expected {
			err = fmt.Errorf("%s: %q requires a minimum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	case maxArgs:
		if context.NArg() > expected {
			err = fmt.Errorf("%s: %q requires a maximum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	}

	if err != nil {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(context, cmdName)
		return err
	}
	return nil
}

func logrusToStderr() bool {
	l, ok := logrus.StandardLogger().Out.(*os.File)
	return ok && l.Fd() == os.Stderr.Fd()
}

func revisePidFile(context *cli.Context) error {
	pidFile := context.String("pid-file")
	if pidFile == "" {
		return nil
	}

	// convert pid-file to an absolute path so we can write to the right
	// file after chdir to bundle
	pidFile, err := filepath.Abs(pidFile)
	if err != nil {
		return err
	}
	return context.Set("pid-file", pidFile)
}

// parseBoolOrAuto returns (nil, nil) if s is empty or "auto"
func parseBoolOrAuto(s string) (*bool, error) {
	if s == "" || strings.ToLower(s) == "auto" {
		return nil, nil
	}
	b, err := strconv.ParseBool(s)
	return &b, err
}

var errEmptyID = errors.New("container id is empty")

func createContainer(context *cli.Context, id string) (container.Container, error) {
	root := context.GlobalString("root")
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	factory, err := container.New(abs)
	if err != nil {
		return nil, err
	}
	return factory.Create(id)
}

func startContainer(context *cli.Context) (int, error) {
	id := context.Args().First()
	if id == "" {
		return -1, errEmptyID
	}

	c, err := createContainer(context, id)
	if err != nil {
		return -1, err
	}
	cmdArray := make([]string, 0)
	for _, arg := range context.Args().Tail() {
		cmdArray = append(cmdArray, arg)
	}
	envs := context.StringSlice("e")
	// todo 传入 cwd
	// cwd := context.StringSlice("cwd")
	process := &container.Process{
		Cwd:  "/",
		Args: cmdArray,
		Env:  envs,
	}
	c.Run(process)
	// process := &container.Process{
	// 	Cwd:  "/",
	// 	Args: cmdArray,
	// 	Env:  envs,
	// }
	// initProcess := container.NewInitProcess(process)
	// containerName := context.String("name")
	// volume := context.String("v")
	// net := context.String("net")

	// ports := context.StringSlice("p")
	// return nil, ni
	// c.Run(initProcess)
	return 0, nil
}
