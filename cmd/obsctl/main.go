package main

import (
	"github.com/eyazici90/obsctl/internal"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := newApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newApp() *cli.App {
	return &cli.App{
		Name:        "obsctl",
		Version:     "v0.0.1",
		Description: "Observability CLI",
		Commands: []*cli.Command{
			rulesCmd,
			metricsCmd,
		},
	}
}

type Config struct {
	*internal.ExportConfig
	*internal.AnalyserConfig
}

func actionSetup(c *cli.Context) *Config {
	addr := c.String("prom-addr")
	limit := c.Uint64("limit")
	out := c.String("output")
	rfile, mfile := c.String("rules-file"), c.String("metrics-file")
	return &Config{
		ExportConfig: &internal.ExportConfig{
			Addr:   addr,
			Output: out,
		},
		AnalyserConfig: &internal.AnalyserConfig{
			RulesFile:   rfile,
			MetricsFile: mfile,
			Limit:       limit,
		},
	}
}
