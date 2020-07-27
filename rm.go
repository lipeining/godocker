// +build linux

package main

import (
	"fmt"

	"github.com/lipeining/godocker/container"
	"github.com/urfave/cli"
)

var rmCommand = cli.Command{
	Name:      "stop",
	Usage:     "stop displays the processes running inside a container",
	ArgsUsage: `<container-id> [stop options]`,
	Action: func(context *cli.Context) error {
		if err := checkArgs(context, 1, minArgs); err != nil {
			return err
		}
		c, err := getContainer(context, context.Args().First())
		if err != nil {
			return err
		}
		fmt.Println(c)
		if err := c.DeleteState(); err != nil {
			return err
		}
		// 删除 workspace
		container.DeleteWorkSpace(c.Name, "")
		return nil
	},
	SkipArgReorder: true,
}
