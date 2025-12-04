package main

import (
	"os"
	"time"

	"fyne.io/fyne/v2/app"
	"github.com/Elisandil/go-snap/internal/ui"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()

	setupLogger()

	log.Info().Msg("Starting GoSnap Desktop Application")

	appID := getEnvOrDefault("DESKTOP_APP_ID", "com.elisandil.gosnap.test")

	a := app.NewWithID(appID)
	a.Settings().SetTheme(&ui.CustomTheme{})

	mainWindow := ui.NewMainWindow(a)
	mainWindow.ShowAndRun()
}

// ------------------------------------------------------------------------------------------------
// 											PRIVATE FUNCTIONS
//-------------------------------------------------------------------------------------------------

// getEnvOrDefault retrieves the value of the environment variable or returns the default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupLogger configures the zerolog logger to output to the console with a specific time format.
func setupLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
