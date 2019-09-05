package main

import (
	"os"

	"github.com/docker/docker/cli/config"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// filled in at compile time
var gitCommit = ""

const (
	version = "1.9.9-dev"
	usage   = "registry client to inspect and push multi-platform OCI & Docker v2 images"
)

func main() {
	if err := runApplication(); err != nil {
		logrus.Errorf("manifest-tool failed with error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runApplication() error {
	app := cli.NewApp()
	app.Name = os.Args[0]
	app.Version = version + " (commit: " + gitCommit + ")"
	app.Usage = usage
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output",
		},
		cli.BoolFlag{
			Name:  "insecure",
			Usage: "allow http/insecure registry communication",
		},
		cli.StringFlag{
			Name:  "username",
			Value: "",
			Usage: "registry username",
		},
		cli.StringFlag{
			Name:  "password",
			Value: "",
			Usage: "registry password",
		},
		cli.StringFlag{
			Name:  "docker-cfg",
			Value: config.Dir(),
			Usage: "Docker's cli config for auth",
		},
	}
	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.WarnLevel)
		}
		return nil
	}
	// currently support inspect and pushml
	app.Commands = []cli.Command{
		inspectCmd,
		pushCmd,
	}

	return app.Run(os.Args)
}
