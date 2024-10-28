package main

import (
	"os"

	"github.com/eyazici90/owl/internal"
	"github.com/urfave/cli/v2"
)

var root = &cli.App{
	Name:        "owl",
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
		IdlerConfig: &internal.IdlerConfig{
			RulesFile:      rfile,
			MetricsFile:    mfile,
			DashboardsFile: dfile,
			Limit:          limit,
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
