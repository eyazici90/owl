package main

import (
	"fmt"
	"log"
	"strings"

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

	log.Printf("rules export finished!")
	return nil
}

func actionRulesAnalyse(c *cli.Context) error {
	cfg := actionSetup(c)
	analyser, err := internal.NewPromRulesAnalyser(cfg.AnalyserConfig)
	if err != nil {
		return fmt.Errorf("new prom analyser: %w", err)
	}
	res, err := analyser.FindRulesMissingMetrics(c.Context)
	if err != nil {
		return fmt.Errorf("rule missing: %w", err)
	}
	for _, v := range res {
		log.Printf("type: %s, rule: %s, missing_metrics: [%s]", v.RuleType, v.Rule, strings.Join(v.Metrics, ","))
	}
	return nil
}
