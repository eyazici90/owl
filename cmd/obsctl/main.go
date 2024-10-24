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
		},
	}
}

type Config struct {
	*internal.RuleExportConfig
	*internal.RuleAnalyserConfig
}

func actionSetup(c *cli.Context) *Config {
	addr := c.String("prom-addr")
	limit := c.Uint64("limit")
	out := c.String("output")
	return &Config{
		RuleExportConfig: &internal.RuleExportConfig{
			Addr:   addr,
			Output: out,
		},
		RuleAnalyserConfig: &internal.RuleAnalyserConfig{
			Addr:  addr,
			Limit: limit,
		},
	}
}
