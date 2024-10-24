package main

import (
	"fmt"
	"log/slog"

	"github.com/eyazici90/obsctl/internal"
	"github.com/urfave/cli/v2"
)

var rulesCmd = &cli.Command{
	Name: "rules",
	Subcommands: []*cli.Command{
		{
			Name:   "export",
			Usage:  `exports prom rules to csv files`,
			Action: actionRulesExport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "output",
					Value: "rules.csv",
				},
			},
		},
		{
			Name:   "analyse",
			Usage:  `Scans prom rules to find rules that are missing metrics`,
			Action: actionRulesAnalyse,
			Flags: []cli.Flag{
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
		{
			Name:   "slowest",
			Action: actionRulesSlowest,
			Usage:  `Scans prom rules to find slowest based on evaluation duration`,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "rules-file",
					Value: "rules.csv",
				},
				&cli.Uint64Flag{
					Name:  "limit",
					Value: 10,
				},
			},
		},
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "prom-addr",
			Value: "https://demo.promlabs.com/",
			// Required: true,
		},
	},
}

func actionRulesExport(c *cli.Context) error {
	cfg := actionSetup(c)
	exp, err := internal.NewRulesExporter(cfg.ExportConfig)
	if err != nil {
		return fmt.Errorf("new prom analyser: %w", err)
	}
	if err = exp.Export(c.Context); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	slog.Info("Rules export finished!")
	return nil
}

func actionRulesAnalyse(c *cli.Context) error {
	cfg := actionSetup(c)
	pra := internal.NewPromRulesAnalyser(cfg.AnalyserConfig)
	res, err := pra.FindRulesMissingMetrics(c.Context)
	if err != nil {
		return fmt.Errorf("rules missing: %w", err)
	}
	slog.Info("Found",
		slog.Int("total", len(res)),
	)
	for _, rule := range res {
		slog.Info("Found",
			slog.String("item", fmt.Sprintf("%+v", rule)),
		)
	}
	return nil
}

func actionRulesSlowest(c *cli.Context) error {
	cfg := actionSetup(c)
	prs := internal.NewPromRulesSlowest(cfg.SlowestConfig)
	res, err := prs.Get(c.Context)
	if err != nil {
		return fmt.Errorf("rules missing: %w", err)
	}
	slog.Info("Found",
		slog.Int("total", len(res)),
	)
	for _, slow := range res {
		slog.Info("Found",
			slog.String("item", fmt.Sprintf("%+v", slow)),
		)
	}
	return nil
}
