package ui

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// customError implements the error interface for displaying custom error messages in dialogs.
type customError struct {
	message string
}

// Error returns the error message.
func (e *customError) Error() string {
	return e.message
}

// ShowErrorDialog displays an error dialog with the provided message.
func ShowErrorDialog(window fyne.Window, message string) {
	dialog.ShowError(
		&customError{
			message: message,
		},
		window,
	)
}

// ShowSuccessDialog displays a success dialog with the provided message.
func ShowSuccessDialog(window fyne.Window, message string) {
	dialog.ShowInformation("Success", message, window)
}

// ShowConfirmDialog displays a confirmation dialog with the provided title and message.
func ShowConfirmDialog(window fyne.Window, title, message string, callback func(bool)) {
	dialog.ShowConfirm(title, message, callback, window)
}

// ShowAboutDialog displays an "About" dialog with application information.
// It includes the app name, version, description, author, and a hyperlink to the GitHub page.
func ShowAboutDialog(window fyne.Window) {
	content := container.NewVBox(
		widget.NewLabel("GoSnap Desktop Client"),
		widget.NewLabel("Version: 1.0"),
		widget.NewSeparator(),
		widget.NewLabel("A modern URL shortener application"),
		widget.NewLabel("Built with Go & Fyne GUI toolkit"),
		widget.NewSeparator(),
		widget.NewLabel("© 2025 GoSnap Team-aogdev"),
		widget.NewHyperlink("Visit my Github", parseURL("https://github.com/Elisandil")),
	)

	d := dialog.NewCustom("About GoSnap", "Close", content, window)
	d.Show()
}

// ---------------------------------------------------------------------------------------------
//                                        PRIVATE METHODS
// ---------------------------------------------------------------------------------------------

// parseURL is a helper function to parse a URL string and panic if it's invalid.
func parseURL(urlStr string) *url.URL {
	u, err := url.Parse(urlStr)
	if err != nil {
		panic("Invalid URL: " + urlStr)
	}
	return u
}
