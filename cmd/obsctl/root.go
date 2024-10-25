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
			Name:  "log-level",
			Value: "info",
		},
	},
}

type Config struct {
	*internal.ExportConfig
	*internal.DashboardsExportConfig
	*internal.CheckerConfig
	*internal.SlowestConfig
	*internal.TopListerConfig
}

func actionSetup(c *cli.Context) *Config {
	level := c.String("log-level")
	l := internal.ParseLevel(level)
	internal.SetUpSlog(os.Stderr, l)

	addr := c.String("addr")
	limit := c.Uint64("limit")
	out := c.String("output")
	rfile := c.String("rules-file")
	mfile := c.String("metrics-file")
	dfile := c.String("dashboards-file")
	token := c.String("svc-token")
	expr := &internal.ExportConfig{
		Addr:   addr,
		Output: out,
	}
	return &Config{
		ExportConfig: expr,
		DashboardsExportConfig: &internal.DashboardsExportConfig{
			ExportConfig: expr,
			SvcToken:     token,
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
		TopListerConfig: &internal.TopListerConfig{
			DashboardFile: dfile,
			Limit:         limit,
		},
	}
}
