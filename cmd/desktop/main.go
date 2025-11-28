package main

import (
	"os"
	"time"

	"fyne.io/fyne/v2/app"
	"github.com/Elisandil/GoSnap/internal/ui"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	setupLogger()

	log.Info().Msg("Starting GoSnap Desktop Application")

	a := app.NewWithID("com.elisandil.gosnap")
	a.Settings().SetTheme(&ui.CustomTheme{})

	mainWindow := ui.NewMainWindow(a)
	mainWindow.ShowAndRun()
}

// setupLogger configures the zerolog logger to output to the console with a specific time format.
func setupLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
