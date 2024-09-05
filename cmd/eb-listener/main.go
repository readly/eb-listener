package main

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/readly/eb-listener/pkg/app"
)

func main() {
	logHandler := tint.NewHandler(os.Stdout, &tint.Options{
		// Level: slog.LevelDebug,
		// AddSource: true,
	})
	log := slog.New(logHandler).With(
		slog.Int("pid", os.Getpid()),
	)
	slog.SetDefault(log)

	if err := (app.CLI).Run(os.Args); err != nil {
		slog.Error("fail", "err", err)
	}
}
