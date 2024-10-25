package main

import (
	"fmt"
	"log/slog"

	"github.com/eyazici90/obsctl/internal"
	"github.com/urfave/cli/v2"
)

var dashboardsCmd = &cli.Command{
	Name: "dashboards",
	Subcommands: []*cli.Command{
		{
			Name:   "export",
			Usage:  `exports grafana dashboards to csv files`,
			Action: actionDashboardsExport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Value:   "dashboards.csv",
				},
			},
		},
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "addr",
			Value: "play.grafana.org",
			// Required: true,
		},
		&cli.StringFlag{
			Name:     "svc-token",
			Required: true,
		},
	},
}

func actionDashboardsExport(c *cli.Context) error {
	cfg := actionSetup(c)
	exp, err := internal.NewDashboardsExporter(cfg.DashboardsExportConfig)
	if err != nil {
		return fmt.Errorf("dashboard exporter: %w", err)
	}
	if err = exp.Export(c.Context); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	slog.Info("Dashboards export finished!")
	return nil
}
