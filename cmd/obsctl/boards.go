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
			Usage:  `exports grafana dashboards to csv file`,
			Action: actionDashboardsExport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Value:   "dashboards.csv",
				},
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
		},
		{
			Name:   "top-metrics",
			Usage:  `Lists metrics that are used most in the grafana dashboards`,
			Action: actionDashboardsTopMetrics,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "dashboards-file",
					Value: "dashboards.csv",
				},
				&cli.Uint64Flag{
					Name:  "limit",
					Value: 10,
				},
			},
		},
		{
			Name:   "idle",
			Usage:  `Find panels in the dashboard whose metrics don't exist anymore'`,
			Action: actionDashboardsIdle,
			Flags:  []cli.Flag{},
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

func actionDashboardsTopMetrics(c *cli.Context) error {
	cfg := actionSetup(c)
	tl := internal.NewTopMetricsLister(cfg.TopListerConfig)
	res, err := tl.List(c.Context)
	if err != nil {
		return fmt.Errorf("list top: %w", err)
	}
	slog.Info("Found",
		slog.Int("total", len(res.Usages)),
		slog.Int("errs-count", len(res.ParseErrs)),
	)
	for _, usage := range res.Usages {
		slog.Info("Usage",
			slog.String("item", fmt.Sprintf("%+v", usage)),
		)
	}
	return nil
}

func actionDashboardsIdle(c *cli.Context) error {
	return nil
}
