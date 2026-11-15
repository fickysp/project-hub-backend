package main

import (
	"log/slog"
	"os"

	"github.com/fickysp/project-hub-backend.git/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		slog.Error("Application stopped", "error", err)
		os.Exit(1)
	}
}