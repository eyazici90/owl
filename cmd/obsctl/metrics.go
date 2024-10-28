package main

import (
	"fmt"
	"log/slog"

	"github.com/eyazici90/obsctl/internal"
	"github.com/urfave/cli/v2"
)

var metricsCmd = &cli.Command{
	Name: "metrics",
	Subcommands: []*cli.Command{
		{
			Name:   "export",
			Usage:  `exports prom metrics to csv file`,
			Action: actionMetricsExport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Value:   "metrics.csv",
				},
				&cli.StringFlag{
					Name:  "addr",
					Value: "https://demo.promlabs.com/",
					// Required: true,
				},
				&cli.StringFlag{
					Name:  "since",
					Value: "720h",
				},
			},
		},
		{
			Name:   "idle",
			Usage:  `Find metrics that are not used in any grafana dashboards & prom rules`,
			Action: actionMetricsIdle,
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

func actionMetricsExport(c *cli.Context) error {
	cfg := actionSetup(c)
	exp, err := internal.NewMetricsExporter(cfg.MetricsExporterConfig)
	if err != nil {
		return fmt.Errorf("new prom analyser: %w", err)
	}
	if err = exp.Export(c.Context); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	slog.Info("Metrics export finished!")
	return nil
}

func actionMetricsIdle(c *cli.Context) error {
	cfg := actionSetup(c)
	mi := internal.NewMetricsIdler(cfg.IdlerConfig)
	res, err := mi.List(c.Context)
	if err != nil {
		return fmt.Errorf("list idle metrics: %w", err)
	}
	for _, pe := range res.ParseErrs {
		slog.Debug("Error", slog.Any("msg", pe))
	}
	for _, ds := range res.IdleMetrics {
		slog.Info("Found",
			slog.String("item", fmt.Sprintf("%+v", ds)),
		)
	}
	slog.Info("Found",
		slog.Int("total", len(res.IdleMetrics)),
		slog.Int("err-count", len(res.ParseErrs)),
	)
	return nil
}
