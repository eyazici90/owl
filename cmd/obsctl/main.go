package main

import (
	"log/slog"
	"os"
)

func main() {
	if err := root.Run(os.Args); err != nil {
		slog.Error("App run completed with error(s)", slog.Any("err", err))
	}
}
