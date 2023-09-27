package main

import (
	"fmt"
	"os"
	scli "static-power/cli"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                 "static-power",
		Suggest:              true,
		EnableBashCompletion: true,
		Version:              "v1.0.51cae2c",
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
		},
	}
	app.Setup()
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		return
	}
}
