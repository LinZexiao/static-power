package main

import (
	"fmt"
	"os"
	scli "static-power/cli"

	"github.com/urfave/cli/v2"
)

var (
	CurrentCommit string

	BuildVersion = "v1.1"

	Version = BuildVersion + CurrentCommit
)

func main() {
	app := &cli.App{
		Name:                 "static-power",
		Suggest:              true,
		EnableBashCompletion: true,
		Version:              Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "listen",
				Aliases: []string{
					"l",
				},
				Value: "127.0.0.1:8090",
				Usage: "listen address",
			},
		},
		Commands: []*cli.Command{
			scli.DaemonCmd,
			scli.UpdatePowerCmd,
			scli.UpdateAgentCmd,
			scli.CheckCmd,
			scli.DiffCmd,
		},
	}
	app.Setup()
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		return
	}
}
