package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"owl": func() int {
			if err := root.Run(os.Args); err != nil {
				slog.Error("App run completed with error(s)", slog.Any("err", err))
				return 1
			}
			return 0
		},
	}))
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "tests",
	})
}
