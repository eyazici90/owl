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
			Name:   "top-used",
			Usage:  `Lists metrics & rules that are used most in the grafana dashboards`,
			Action: actionDashboardsTopUsed,
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "dashboards-file",
					Value: "dashboards.csv",
				},
				&cli.StringFlag{
					Name:  "rules-file",
					Value: "rules.csv",
				},
				&cli.StringFlag{
					Name:  "metrics-file",
					Value: "metrics.csv",
				},
				&cli.Uint64Flag{
					Name:  "limit",
					Value: 10,
				},
			},
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

func actionDashboardsTopUsed(c *cli.Context) error {
	cfg := actionSetup(c)
	tl := internal.NewTopUsedListerInGrafana(cfg.TopListerConfig)
	res, err := tl.List(c.Context)
	if err != nil {
		return fmt.Errorf("list top: %w", err)
	}
	for _, pe := range res.ParseErrs {
		slog.Debug("Error", slog.Any("msg", pe))
	}
	for _, usage := range res.Usages {
		slog.Info("Usage",
			slog.String("item", fmt.Sprintf("%+v", usage)),
		)
	}
	slog.Info("Found",
		slog.Int("total", len(res.Usages)),
		slog.Int("err-count", len(res.ParseErrs)),
	)
	return nil
}

func actionDashboardsIdle(c *cli.Context) error {
	cfg := actionSetup(c)
	dsi := internal.NewDashboardsIdler(cfg.IdlerConfig)
	res, err := dsi.List(c.Context)
	if err != nil {
		return fmt.Errorf("list idle dashboards: %w", err)
	}
	for _, pe := range res.ParseErrs {
		slog.Debug("Error", slog.Any("msg", pe))
	}
	for _, ds := range res.IdleDashboards {
		slog.Info("Found",
			slog.String("item", fmt.Sprintf("%+v", ds)),
		)
	}
	slog.Info("Found",
		slog.Int("total", len(res.IdleDashboards)),
		slog.Int("err-count", len(res.ParseErrs)),
	)
	return nil
}
