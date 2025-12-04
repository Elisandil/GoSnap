package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type MainWindow struct {
	app    fyne.App
	window fyne.Window
	client *APIClient
	tabs   *container.AppTabs

	createTab   *CreateTab
	statsTab    *StatsTab
	historyTab  *HistoryTab
	settingsTab *SettingsTab
}

func NewMainWindow(app fyne.App) *MainWindow {
	return NewMainWindowWithClient(app, NewAPIClient("http://localhost:8080"))
}

// NewMainWindowWithClient creates a new main window with a custom API client.
func NewMainWindowWithClient(app fyne.App, client *APIClient) *MainWindow {
	w := &MainWindow{
		app:    app,
		window: app.NewWindow("GoSnap - URL Shortener"),
		client: client,
	}

	w.setupUI()
	w.window.Resize(fyne.NewSize(450, 500))
	w.window.CenterOnScreen()

	return w
}

// ---------------------------------------------------------------------------------------------
//                                      PRIVATE METHODS
// ---------------------------------------------------------------------------------------------

// setupUI initializes the main window UI components.
func (w *MainWindow) setupUI() {
	w.createTab = NewCreateTab(w.client, w.onURLCreated)
	w.statsTab = NewStatsTab(w.client)
	w.historyTab = NewHistoryTab(w.client)
	w.settingsTab = NewSettingsTab(w.client, w.onSettingsChanged)

	w.tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Create", fyne.CurrentApp().Settings().Theme().Icon("contentAdd"),
			w.createTab.Build()),
		container.NewTabItemWithIcon("Stats", fyne.CurrentApp().Settings().Theme().Icon("info"),
			w.statsTab.Build()),
		container.NewTabItemWithIcon("History", fyne.CurrentApp().Settings().Theme().Icon("history"),
			w.historyTab.Build()),
		container.NewTabItemWithIcon("Settings", fyne.CurrentApp().Settings().Theme().Icon("settings"),
			w.settingsTab.Build()),
	)

	w.tabs.SetTabLocation(container.TabLocationTop)

	w.window.SetContent(w.tabs)
	w.window.SetMainMenu(w.makeMenu())
}

// makeMenu creates the main menu for the application.
func (w *MainWindow) makeMenu() *fyne.MainMenu {
	aboutItem := fyne.NewMenuItem("About", func() {
		ShowAboutDialog(w.window)
	})

	quitItem := fyne.NewMenuItem("Quit", func() {
		w.app.Quit()
	})

	fileMenu := fyne.NewMenu("File", quitItem)
	helpMenu := fyne.NewMenu("Help", aboutItem)

	return fyne.NewMainMenu(fileMenu, helpMenu)
}

// onURLCreated is called when a new short URL is created.
func (w *MainWindow) onURLCreated(shortCode string) {
	w.historyTab.Refresh()
	w.tabs.SelectIndex(2)
	ShowSuccessDialog(w.window, "Short URL created: "+shortCode)
}

// onSettingsChanged is called when settings are updated.
func (w *MainWindow) onSettingsChanged(baseURL string) {
	w.client.SetBaseURL(baseURL)
	ShowSuccessDialog(w.window, "Settings updated successfully")
}

// ShowAndRun displays the main window and starts the application event loop.
func (w *MainWindow) ShowAndRun() {
	w.window.ShowAndRun()
}
