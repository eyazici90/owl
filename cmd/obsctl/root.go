package main

import (
	"github.com/eyazici90/obsctl/internal"
	"github.com/urfave/cli/v2"
	"os"
)

var root = &cli.App{
	Name:        "obsctl",
	Version:     "v0.0.1",
	Description: "Observability CLI",
	Commands: []*cli.Command{
		rulesCmd,
		metricsCmd,
		dashboardsCmd,
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "log-level",
		},
	},
}

type Config struct {
	*internal.ExportConfig
	*internal.CheckerConfig
	*internal.SlowestConfig
}

func actionSetup(c *cli.Context) *Config {
	level := c.String("log-level")
	l := internal.ParseLevel(level)
	internal.SetUpSlog(os.Stderr, l)

	addr := c.String("addr")
	limit := c.Uint64("limit")
	out := c.String("output")
	rfile, mfile := c.String("rules-file"), c.String("metrics-file")
	return &Config{
		ExportConfig: &internal.ExportConfig{
			Addr:   addr,
			Output: out,
		},
		CheckerConfig: &internal.CheckerConfig{
			RulesFile:   rfile,
			MetricsFile: mfile,
			Limit:       limit,
		},
		SlowestConfig: &internal.SlowestConfig{
			RulesFile: rfile,
			Limit:     limit,
		},
	}
}
