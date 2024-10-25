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
			},
		},
	},
}

func actionMetricsExport(c *cli.Context) error {
	cfg := actionSetup(c)
	exp, err := internal.NewMetricsExporter(cfg.ExportConfig)
	if err != nil {
		return fmt.Errorf("new prom analyser: %w", err)
	}
	if err = exp.Export(c.Context); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	slog.Info("Metrics export finished!")
	return nil
}
