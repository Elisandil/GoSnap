package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type URLHistoryItem struct {
	ShortCode string
	ShortURL  string
	LongURL   string
	CreatedAt time.Time
}

type HistoryTab struct {
	client           *APIClient
	history          []URLHistoryItem
	list             *widget.List
	emptyLabel       *widget.Label
	contentContainer *fyne.Container
}

func NewHistoryTab(client *APIClient) *HistoryTab {
	return &HistoryTab{
		client:  client,
		history: make([]URLHistoryItem, 0),
	}
}

// Build constructs the UI for the History tab.
func (t *HistoryTab) Build() fyne.CanvasObject {
	t.list = t.createHistoryList()
	t.emptyLabel = t.createEmptyLabel()
	toolbar := t.createToolbar()

	t.contentContainer = container.NewStack()
	t.updateVisibility()

	header := container.NewVBox(
		widget.NewLabelWithStyle("URL History", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("List of all created short URLs in this session"),
		toolbar,
		widget.NewSeparator(),
	)

	content := container.NewVBox(
		header,
		t.contentContainer,
	)

	return content
}

// AddItem adds a new URL history item to the list.
func (t *HistoryTab) AddItem(shortCode, shortURL, longURL string) {
	item := URLHistoryItem{
		ShortCode: shortCode,
		ShortURL:  shortURL,
		LongURL:   longURL,
		CreatedAt: time.Now(),
	}

	t.history = append([]URLHistoryItem{item}, t.history...)
	t.list.Refresh()
	t.updateVisibility()
}

// Refresh refreshes the history list display.
func (t *HistoryTab) Refresh() {
	t.list.Refresh()
}

//---------------------------------------------------------------------------------------------
//                                        PRIVATE METHODS
//---------------------------------------------------------------------------------------------

// createHistoryList creates the list widget for displaying URL history.
func (t *HistoryTab) createHistoryList() *widget.List {
	return widget.NewList(
		func() int {
			return len(t.history)
		},
		func() fyne.CanvasObject {
			return t.createListItemTemplate()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t.updateListItem(id, obj)
		},
	)
}

// createListItemTemplate creates a template for each list item.
func (t *HistoryTab) createListItemTemplate() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewCard("", "",
			container.NewVBox(
				widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{
					Bold: true,
				}),
				widget.NewLabel(""),
				widget.NewLabel(""),
				container.NewHBox(
					widget.NewButton("Copy", nil),
					widget.NewButton("Open", nil),
				),
			),
		),
	)
}

// updateListItem updates a list item with data from the history.
func (t *HistoryTab) updateListItem(id widget.ListItemID, obj fyne.CanvasObject) {

	if id >= len(t.history) {
		return
	}

	item := t.history[id]
	card := obj.(*fyne.Container).Objects[0].(*widget.Card)
	content := card.Content.(*fyne.Container)

	card.SetTitle(fmt.Sprintf("Short Code: %s", item.ShortCode))

	t.updateLabels(content, item)
	t.updateButtons(content, item)
}

// updateLabels updates the labels in a list item.
func (t *HistoryTab) updateLabels(content *fyne.Container, item URLHistoryItem) {
	shortURLLabel := content.Objects[0].(*widget.Label)
	shortURLLabel.SetText(item.ShortURL)

	longURLLabel := content.Objects[1].(*widget.Label)
	longURLLabel.SetText(fmt.Sprintf("Long URL: %s", item.LongURL))

	dateLabel := content.Objects[2].(*widget.Label)
	dateLabel.SetText(fmt.Sprintf("Created: %s", item.CreatedAt.Format("02/01/2006 15:04:05")))
}

// updateButtons configures the action buttons for a list item.
func (t *HistoryTab) updateButtons(content *fyne.Container, item URLHistoryItem) {
	buttons := content.Objects[3].(*fyne.Container)

	copyBtn := buttons.Objects[0].(*widget.Button)
	copyBtn.OnTapped = func() {
		t.handleCopy(item.ShortURL)
	}

	openBtn := buttons.Objects[1].(*widget.Button)
	openBtn.OnTapped = func() {
		t.handleOpen(item.ShortURL)
	}
}

// createToolbar creates the toolbar with refresh and clear buttons.
func (t *HistoryTab) createToolbar() *fyne.Container {
	refreshBtn := widget.NewButton("Refresh", func() {
		t.Refresh()
	})

	clearBtn := widget.NewButton("Clear History", t.handleClear)
	clearBtn.Importance = widget.DangerImportance

	return container.NewHBox(refreshBtn, clearBtn)
}

// createEmptyLabel creates the label displayed when history is empty.
func (t *HistoryTab) createEmptyLabel() *widget.Label {
	emptyLabel := widget.NewLabel("No URLs in history yet. Create some short URLs to see them here!")
	emptyLabel.Wrapping = fyne.TextWrapOff
	emptyLabel.Alignment = fyne.TextAlignCenter

	return emptyLabel
}

// handleCopy copies the short URL to clipboard.
func (t *HistoryTab) handleCopy(shortURL string) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	window.Clipboard().SetContent(shortURL)
	ShowSuccessDialog(window, "Copied to clipboard!")
}

// handleOpen opens the short URL in the default browser.
func (t *HistoryTab) handleOpen(shortURL string) {
	err := fyne.CurrentApp().OpenURL(parseURL(shortURL))
	if err != nil {
		window := fyne.CurrentApp().Driver().AllWindows()[0]
		ShowErrorDialog(window, "Failed to open URL: "+err.Error())
	}
}

// handleClear clears the history list.
func (t *HistoryTab) handleClear() {
	t.history = make([]URLHistoryItem, 0)
	t.list.Refresh()
	t.updateVisibility()
}

// updateVisibility updates the visibility of the list and empty label.
func (t *HistoryTab) updateVisibility() {
	if len(t.history) == 0 {
		t.contentContainer.Objects = []fyne.CanvasObject{container.NewCenter(t.emptyLabel)}
	} else {
		t.contentContainer.Objects = []fyne.CanvasObject{t.list}
	}
	t.contentContainer.Refresh()
}
