package main

import (
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
