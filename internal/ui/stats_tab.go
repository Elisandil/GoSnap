package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/rs/zerolog/log"
)

type StatsTab struct {
	client *APIClient

	shortCodeEntry *widget.Entry
	searchBtn      *widget.Button
	statsCard      *widget.Card
	statsContent   *fyne.Container
}

func NewStatsTab(client *APIClient) *StatsTab {
	return &StatsTab{
		client: client,
	}
}

// Build constructs the Stats tab UI.
func (t *StatsTab) Build() fyne.CanvasObject {
	searchForm := t.createSearchForm()
	t.statsContent = container.NewVBox()
	t.statsCard = t.createStatsCard()

	return container.NewPadded(
		container.NewVBox(
			searchForm,
			t.statsCard,
		),
	)
}

//---------------------------------------------------------------------------------------------
//                                      PRIVATE METHODS
//---------------------------------------------------------------------------------------------

// createSearchForm creates the search form for entering short codes.
func (t *StatsTab) createSearchForm() *widget.Card {
	t.shortCodeEntry = widget.NewEntry()
	t.shortCodeEntry.SetPlaceHolder("Enter short code (e.g., abc123)")

	t.searchBtn = widget.NewButton("Get Statistics", t.handleGetStats)
	t.searchBtn.Importance = widget.HighImportance

	return widget.NewCard("Search Statistics", "Enter a short code to view its statistics",
		container.NewVBox(
			t.shortCodeEntry,
			t.searchBtn,
		),
	)
}

// createStatsCard creates the card for displaying statistics.
func (t *StatsTab) createStatsCard() *widget.Card {
	card := widget.NewCard("Statistics", "", t.statsContent)
	card.Hide()
	return card
}

// setButtonLoading updates the button state during loading.
func (t *StatsTab) setButtonLoading(loading bool) {
	if loading {
		t.searchBtn.Disable()
		t.searchBtn.SetText("Loading...")
	} else {
		t.searchBtn.Enable()
		t.searchBtn.SetText("Get Statistics")
	}
}

// handleGetStats handles the logic for retrieving statistics based on the short code.
func (t *StatsTab) handleGetStats() {
	shortCode := t.shortCodeEntry.Text

	if shortCode == "" {
		ShowErrorDialog(fyne.CurrentApp().Driver().AllWindows()[0], "Please enter a short code")
		return
	}

	t.searchBtn.Disable()
	t.searchBtn.SetText("Loading...")

	go func() {
		stats, err := t.client.GetStats(shortCode)

		if err != nil {
			log.Error().Err(err).Msg("Error getting statistics")
			ShowErrorDialog(fyne.CurrentApp().Driver().AllWindows()[0], "Failed to get statistics: "+err.Error())
			t.searchBtn.Enable()
			t.searchBtn.SetText("Get Statistics")
			return
		}

		fyne.Do(func() {
			t.displayStats(stats.ShortCode, stats.LongURL, stats.Clicks, stats.CreatedAt.Format("02/01/2006 15:04:05"))

			t.searchBtn.Enable()
			t.searchBtn.SetText("Get Statistics")
		})
	}()
}

// displayStats updates the UI to show the retrieved statistics.
func (t *StatsTab) displayStats(shortCode, longURL string, clicks int64, createdAt string) {
	t.statsContent.Objects = []fyne.CanvasObject{
		widget.NewForm(
			widget.NewFormItem("Short Code:", widget.NewLabel(shortCode)),
			widget.NewFormItem("Long URL:", widget.NewLabel(longURL)),
			widget.NewFormItem("Total Clicks:", widget.NewLabel(fmt.Sprintf("%d", clicks))),
			widget.NewFormItem("Created At:", widget.NewLabel(createdAt)),
		),
	}

	t.statsCard.Show()
	t.statsContent.Refresh()
}
