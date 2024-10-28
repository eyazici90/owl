package main

import (
	"fmt"
	"log/slog"

	"github.com/eyazici90/owl/internal"
	"github.com/urfave/cli/v2"
)

var rulesCmd = &cli.Command{
	Name: "rules",
	Subcommands: []*cli.Command{
		{
			Name:   "export",
			Usage:  `Exports prom rules to csv file`,
			Action: actionRulesExport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Value:   "rules.csv",
				},
				&cli.StringFlag{
					Name:  "addr",
					Value: "https://demo.promlabs.com/",
					// Required: true,
				},
			},
		},
		{
			Name:   "idle",
			Usage:  `Scans prom rules to find ones that are missing metrics`,
			Action: actionRulesIdle,
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
			Usage:  `Scans prom rules to find slowest ones based on evaluation durations`,
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
}

func actionRulesExport(c *cli.Context) error {
	cfg := actionSetup(c)
	exp, err := internal.NewRulesExporter(cfg.ExportConfig)
	if err != nil {
		return fmt.Errorf("new rules exporter: %w", err)
	}
	if err = exp.Export(c.Context); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	slog.Info("Rules export finished!")
	return nil
}

func actionRulesIdle(c *cli.Context) error {
	cfg := actionSetup(c)
	pri := internal.NewPromRulesIdler(cfg.IdlerConfig)
	res, err := pri.List(c.Context)
	if err != nil {
		return fmt.Errorf("list idle rules: %w", err)
	}
	for _, rule := range res {
		slog.Info("Found",
			slog.String("item", fmt.Sprintf("%+v", rule)),
		)
	}
	slog.Info("Found",
		slog.Int("total", len(res)),
	)
	return nil
}

func actionRulesSlowest(c *cli.Context) error {
	cfg := actionSetup(c)
	prs := internal.NewPromRulesSlowest(cfg.SlowestConfig)
	res, err := prs.Get(c.Context)
	if err != nil {
		return fmt.Errorf("get slowest: %w", err)
	}
	for _, pe := range res.ParseErrs {
		slog.Debug("Error", slog.Any("msg", pe))
	}
	for _, slow := range res.Rules {
		slog.Info("Slow",
			slog.String("item", fmt.Sprintf("%+v", slow)),
		)
	}
	slog.Info("Found",
		slog.Int("total", len(res.Rules)),
		slog.Int("err-count", len(res.ParseErrs)),
	)
	return nil
}
