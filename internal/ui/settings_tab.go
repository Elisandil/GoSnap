package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/rs/zerolog/log"
)

type SettingsTab struct {
	client    *APIClient
	onChanged func(string)

	serverURLEntry *widget.Entry
	saveBtn        *widget.Button
	testBtn        *widget.Button
	statusLabel    *widget.Label
}

func NewSettingsTab(client *APIClient, onChanged func(string)) *SettingsTab {
	return &SettingsTab{
		client:    client,
		onChanged: onChanged,
	}
}

// Build constructs the settings tab UI.
func (t *SettingsTab) Build() fyne.CanvasObject {
	t.initializeComponents()

	serverCard := t.createServerCard()
	infoCard := t.createInfoCard()

	return container.NewPadded(
		container.NewVBox(
			serverCard,
			infoCard,
		),
	)
}

//--------------------------------------------------------------------------------------------------
//                                   	PRIVATE METHODS
//--------------------------------------------------------------------------------------------------

// initializeComponents initializes all UI components.
func (t *SettingsTab) initializeComponents() {
	t.serverURLEntry = t.createServerURLEntry()
	t.saveBtn = t.createSaveButton()
	t.testBtn = t.createTestButton()
	t.statusLabel = t.createStatusLabel()
}

// createServerURLEntry creates the server URL input field.
func (t *SettingsTab) createServerURLEntry() *widget.Entry {
	entry := widget.NewEntry()
	entry.SetText(t.client.GetBaseURL())
	entry.SetPlaceHolder("http://localhost:8080")
	return entry
}

// createSaveButton creates the save settings button.
func (t *SettingsTab) createSaveButton() *widget.Button {
	btn := widget.NewButton("Save Settings", t.handleSave)
	btn.Importance = widget.HighImportance
	return btn
}

// createTestButton creates the test connection button.
func (t *SettingsTab) createTestButton() *widget.Button {
	return widget.NewButton("Test Connection", t.handleTest)
}

// createStatusLabel creates the status label.
func (t *SettingsTab) createStatusLabel() *widget.Label {
	label := widget.NewLabel("")
	label.Wrapping = fyne.TextWrapWord
	return label
}

// createServerCard creates the server configuration card.
func (t *SettingsTab) createServerCard() *widget.Card {
	settingsForm := widget.NewForm(
		widget.NewFormItem("Server URL:", t.serverURLEntry),
	)

	buttons := container.NewHBox(
		t.saveBtn,
		t.testBtn,
	)

	return widget.NewCard("Server Configuration", "Configure the GoSnap server URL",
		container.NewVBox(
			settingsForm,
			layout.NewSpacer(),
			buttons,
			t.statusLabel,
		),
	)
}

// createInfoCard creates the application information card.
func (t *SettingsTab) createInfoCard() *widget.Card {
	return widget.NewCard("Application Information", "",
		container.NewVBox(
			widget.NewLabel("GoSnap Desktop Client"),
			widget.NewLabel("Version: 1.0.0"),
			widget.NewLabel("A modern URL shortener application"),
			layout.NewSpacer(),
			widget.NewHyperlink("GitHub Repository", parseURL("https://github.com/Elisandil/GoSnap")),
		),
	)
}

func (t *SettingsTab) handleSave() {
	newURL := t.serverURLEntry.Text

	if newURL == "" {
		ShowErrorDialog(fyne.CurrentApp().Driver().AllWindows()[0], "Server URL cannot be empty")
		return
	}

	if t.onChanged != nil {
		t.onChanged(newURL)
	}

	t.statusLabel.SetText("✓ Settings saved successfully")
}

func (t *SettingsTab) handleTest() {
	t.testBtn.Disable()
	t.testBtn.SetText("Testing...")
	t.statusLabel.SetText("Testing connection...")

	go func() {
		err := t.client.HealthCheck()

		if err != nil {
			log.Error().Err(err).Msg("Health check failed")
			t.statusLabel.SetText("✗ Connection failed: " + err.Error())
			ShowErrorDialog(fyne.CurrentApp().Driver().AllWindows()[0], "Connection test failed: "+err.Error())
		} else {
			t.statusLabel.SetText("✓ Connection successful!")
			ShowSuccessDialog(fyne.CurrentApp().Driver().AllWindows()[0], "Server is reachable!")
		}

		t.testBtn.Enable()
		t.testBtn.SetText("Test Connection")
	}()
}
