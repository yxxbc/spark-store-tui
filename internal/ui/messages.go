package ui

import "github.com/yxxbc/spark-store-tui/internal/domain"

type appsLoadedMsg struct {
	sourceID string
	apps     []domain.App
	err      error
}

type categoriesLoadedMsg struct {
	sourceID   string
	categories []domain.Category
	err        error
}

type previewLoadedMsg struct {
	appID      string
	icon       string
	screenshot string
	request    int
}

type previewDueMsg struct {
	appID         string
	iconURL       string
	screenshotURL string
	request       int
}

type downloadFinishedMsg struct {
	task domain.DownloadTask
	err  error
}

type uninstallFinishedMsg struct{ err error }
type installFinishedMsg struct{ err error }
type updateFinishedMsg struct{ err error }

type previewFinishedMsg struct{ err error }

type packageStatusMsg struct {
	appID  string
	status domain.PackageStatus
	err    error
}
