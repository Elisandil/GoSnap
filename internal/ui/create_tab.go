package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/rs/zerolog/log"
)

type CreateTab struct {
	client    *APIClient
	onCreated func(string)

	urlEntry      *widget.Entry
	resultCard    *widget.Card
	shortURLLabel *widget.Label
	shortenBtn    *widget.Button
	copyBtn       *widget.Button
	openBtn       *widget.Button
}

func NewCreateTab(client *APIClient, onCreated func(string)) *CreateTab {
	return &CreateTab{
		client:    client,
		onCreated: onCreated,
	}
}

// Build constructs the UI components for the CreateTab.
// It returns a fyne.CanvasObject that can be added to the main application window.
func (t *CreateTab) Build() fyne.CanvasObject {
	t.urlEntry = t.createURLEntry()
	t.shortenBtn = t.createShortenButton()
	t.resultCard = t.createResultCard()

	form := container.NewVBox(
		widget.NewCard("Create Short URL", "Enter the long URL you want to shorten", container.NewVBox(
			t.urlEntry,
			t.shortenBtn,
		)),
		t.resultCard,
	)

	return container.NewPadded(
		container.NewBorder(nil, nil, nil, nil, form),
	)
}

//---------------------------------------------------------------------------------------------
//                                        PRIVATE METHODS
//---------------------------------------------------------------------------------------------

// createURLEntry creates the entry field for URL input.
func (t *CreateTab) createURLEntry() *widget.Entry {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter the URL to shorten")
	entry.MultiLine = false
	return entry
}

// createShortenButton creates the main button to shorten URLs.
func (t *CreateTab) createShortenButton() *widget.Button {
	btn := widget.NewButton("Shorten URL", t.handleShorten)
	btn.Importance = widget.HighImportance
	return btn
}

// createResultCard creates the card that displays the shortened URL result.
func (t *CreateTab) createResultCard() *widget.Card {
	t.shortURLLabel = widget.NewLabel("")
	t.shortURLLabel.Wrapping = fyne.TextWrapWord

	t.copyBtn = widget.NewButton("Copy to Clipboard", t.handleCopy)
	t.copyBtn.Importance = widget.SuccessImportance

	t.openBtn = widget.NewButton("Open Short URL", t.handleOpen)

	resultBtns := container.NewHBox(t.copyBtn, t.openBtn)

	card := widget.NewCard("", "", container.NewVBox(
		widget.NewLabelWithStyle("Short URL:", fyne.TextAlignLeading, fyne.TextStyle{
			Bold: true,
		}),
		t.shortURLLabel,
		layout.NewSpacer(),
		resultBtns,
	))
	card.Hide()

	return card
}

// handleShorten is called when the shorten button is clicked.
func (t *CreateTab) handleShorten() {
	longURL := t.urlEntry.Text
	if longURL == "" {
		ShowErrorDialog(fyne.CurrentApp().Driver().AllWindows()[0], "Please enter a URL to shorten.")
		return
	}

	t.shortenBtn.Disable()
	t.shortenBtn.SetText("Shortening ...")

	go func() {
		result, err := t.client.CreateShortURL(longURL)
		if err != nil {
			log.Error().Err(err).Msg("Failed to shorten URL")
			ShowErrorDialog(fyne.CurrentApp().Driver().AllWindows()[0], "Failed to shorten URL: "+err.Error())
			t.shortenBtn.Enable()
			t.shortenBtn.SetText("Shorten URL")
			return
		}

		fyne.Do(func() {
			t.shortURLLabel.SetText(result.ShortURL)
			t.resultCard.Show()
			t.urlEntry.SetText("")
			if t.onCreated != nil {
				t.onCreated(result.ShortCode)
			}
			t.shortenBtn.Enable()
			t.shortenBtn.SetText("Shorten URL")

			ShowSuccessDialog(fyne.CurrentApp().Driver().AllWindows()[0], "URL shortened successfully!")
		})
	}()
}

// handleCopy is called when the copy button is clicked.
func (t *CreateTab) handleCopy() {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	window.Clipboard().SetContent(t.shortURLLabel.Text)

	ShowSuccessDialog(window, "Short URL copied to clipboard!")
}

// handleOpen is called when the open button is clicked.
func (t *CreateTab) handleOpen() {
	url := t.shortURLLabel.Text
	if url != "" {
		err := fyne.CurrentApp().OpenURL(parseURL(url))
		if err != nil {
			log.Error().Err(err).Msg("Failed to open URL")
			return
		}
	}
}
