// +build linux

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var psCommand = cli.Command{
	Name:      "ps",
	Usage:     "ps displays the processes running inside a container",
	ArgsUsage: `<container-id> [ps options]`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format, f",
			Value: "table",
			Usage: `select one of: ` + formatOptions,
		},
	},
	Action: func(context *cli.Context) error {
		if err := checkArgs(context, 1, minArgs); err != nil {
			return err
		}
		c, err := getContainer(context, context.Args().First())
		if err != nil {
			return err
		}
		fmt.Println(c)
		containerState, err := c.CurrentState()
		if err != nil {
			return err
		}
		containerStatus, err := c.CurrentStatus()
		if err != nil {
			return err
		}
		switch context.String("format") {
		case "table":
		case "json":
			if err := json.NewEncoder(os.Stdout).Encode(containerState); err != nil {
				return err
			}
			if err := json.NewEncoder(os.Stdout).Encode(containerStatus.String()); err != nil {
				return err
			}
		default:
			return errors.New("invalid format option")
		}
		return nil
	},
	SkipArgReorder: true,
}
