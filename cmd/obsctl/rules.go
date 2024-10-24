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
			Action: actionExport,
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
			Action: actionAnalyse,
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

func actionExport(c *cli.Context) error {
	cfg := actionSetup(c)
	exporter, err := internal.NewRuleExporter(cfg.RuleExportConfig)
	if err != nil {
		return fmt.Errorf("new prom analyser: %w", err)
	}
	if err = exporter.Export(c.Context); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	log.Printf("export finished!")
	return nil
}

func actionAnalyse(c *cli.Context) error {
	cfg := actionSetup(c)
	analyser, err := internal.NewPromRuleAnalyser(cfg.RuleAnalyserConfig)
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
