package main

import (
	"os"

	"github.com/eyazici90/obsctl/internal"
	"github.com/urfave/cli/v2"
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
	*internal.MetricsExporterConfig
	*internal.DashboardsExportConfig
	*internal.IdlerConfig
	*internal.DashboardsIdlerConfig
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
	since := c.String("since")
	rfile := c.String("rules-file")
	mfile := c.String("metrics-file")
	dfile := c.String("dashboards-file")
	token := c.String("svc-token")
	expr := &internal.ExportConfig{
		Addr:   addr,
		Output: out,
	}
	idle := &internal.IdlerConfig{
		RulesFile:   rfile,
		MetricsFile: mfile,
		Limit:       limit,
	}
	return &Config{
		ExportConfig: expr,
		MetricsExporterConfig: &internal.MetricsExporterConfig{
			ExportConfig: expr,
			Since:        since,
		},
		DashboardsExportConfig: &internal.DashboardsExportConfig{
			ExportConfig: expr,
			SvcToken:     token,
		},
		IdlerConfig: idle,
		DashboardsIdlerConfig: &internal.DashboardsIdlerConfig{
			IdlerConfig:    idle,
			DashboardsFile: dfile,
		},
		SlowestConfig: &internal.SlowestConfig{
			RulesFile: rfile,
			Limit:     limit,
		},
		TopListerConfig: &internal.TopListerConfig{
			DashboardsFile: dfile,
			Limit:          limit,
		},
	}
}
