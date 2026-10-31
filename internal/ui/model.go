package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/yxxbc/spark-store-tui/internal/domain"
	"github.com/yxxbc/spark-store-tui/internal/preview"
)

type focusPane int

const (
	focusSources focusPane = iota
	focusApps
)

// Service is the narrow application boundary the UI consumes. It is defined
// here, at the consumer, so the UI never depends on catalog, package or
// download implementations.
type Service interface {
	Sources() []domain.CatalogSource
	Host() domain.PlatformInfo
	ListCategories(context.Context, string) ([]domain.Category, error)
	ListApps(context.Context, string, string) ([]domain.App, error)
	UsesAPM() bool
	ValidateAPM(domain.App) error
	ValidateDownload(domain.App) error
	ExistingPackage(domain.App) string
	PrepareDownload(domain.App) (domain.DownloadPlan, error)
	Install(domain.App, string) (domain.CommandPlan, error)
	Update(domain.App) (domain.CommandPlan, error)
	Uninstall(domain.App) (domain.CommandPlan, error)
	CanUninstall(domain.App) error
	DeletePackage(string) error
	PackageStatus(context.Context, domain.App) (domain.PackageStatus, error)
	ImagePreviewAvailable() bool
	BootstrapCommand() (string, bool)
	PreviewCommand(domain.App) (domain.Command, error)
}

// Terminal owns process handover so the UI never imports os/exec.
type Terminal interface {
	Exec(domain.Command, func(error) tea.Msg) tea.Cmd
}

// Model renders the catalog and drives package operations.
type Model struct {
	service             Service
	terminal            Terminal
	sources             []domain.CatalogSource
	apps                []domain.App
	host                domain.PlatformInfo
	focus               focusPane
	selectedSource      int
	selectedApp         int
	category            string
	categories          []domain.Category
	categoryIndex       int
	width               int
	height              int
	status              string
	loadError           string
	loading             bool
	imageReady          bool
	iconImage           string
	screenshotImage     string
	previewRequest      int
	loadedAt            time.Time
	pendingAction       string
	pendingAppID        string
	searching           bool
	query               string
	descriptionExpanded bool
	pendingPackage      string
	downloading         bool
	checkingPackage     bool
	packageStatus       domain.PackageStatus
	packageStatusAppID  string
}

func New(service Service, terminal Terminal) Model {
	model := Model{
		service:  service,
		terminal: terminal,
		category: "development",
		focus:    focusApps,
		width:    120,
		height:   38,
		status:   "正在加载星火商店目录…",
	}
	if service != nil {
		model.sources = service.Sources()
		model.host = service.Host()
		model.imageReady = service.ImagePreviewAvailable()
	}
	return model
}

// WithRecoveredDownloads surfaces restart recovery without automatically
// restarting network work. Any interrupted package can be resumed with D.
func (m Model) WithRecoveredDownloads(tasks []domain.DownloadTask) Model {
	if len(tasks) == 0 {
		return m
	}
	completed := 0
	interrupted := 0
	for _, task := range tasks {
		switch task.Status {
		case domain.TaskCompleted:
			completed++
		case domain.TaskInterrupted:
			interrupted++
		}
	}
	parts := make([]string, 0, 2)
	if interrupted > 0 {
		parts = append(parts, fmt.Sprintf("已恢复 %d 个中断下载，按 D 可继续", interrupted))
	}
	if completed > 0 {
		parts = append(parts, fmt.Sprintf("已确认 %d 个已完成安装包", completed))
	}
	if len(parts) > 0 {
		m.status = strings.Join(parts, "；")
	}
	return m
}

func (m Model) Init() tea.Cmd { return m.loadCategories() }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case categoriesLoadedMsg:
		if message.sourceID != m.selectedSourceValue().ID {
			return m, nil
		}
		if message.err != nil {
			m.loading = false
			m.loadError = message.err.Error()
			m.status = "分类加载失败"
			return m, nil
		}
		m.categories = message.categories
		m.categoryIndex = categoryIndex(m.categories, m.category)
		if len(m.categories) > 0 {
			m.category = m.categories[m.categoryIndex].ID
		}
		m.status = "正在加载真实应用目录…"
		return m, m.loadApps()
	case appsLoadedMsg:
		if message.sourceID != m.selectedSourceValue().ID {
			return m, nil
		}
		m.loading = false
		m.apps = message.apps
		m.selectedApp = 0
		if m.pendingAppID != "" {
			for index, app := range m.filteredApps() {
				if app.ID == m.pendingAppID {
					m.selectedApp = index
					break
				}
			}
		}
		m.clearPackageStatus()
		m.previewRequest++
		if message.err != nil {
			m.pendingAction = ""
			m.pendingAppID = ""
			m.loadError = message.err.Error()
			m.status = "目录加载失败"
			return m, nil
		}
		m.loadError = ""
		m.loadedAt = time.Now()
		m.status = fmt.Sprintf("已加载 %d 个真实应用（%s）", len(m.apps), m.category)
		if m.pendingAction != "" {
			action := m.pendingAction
			m.pendingAction = ""
			m.pendingAppID = ""
			updated, command := m.handleKey(action)
			return updated, tea.Batch(m.schedulePreview(), command)
		}
		return m, m.schedulePreview()
	case previewDueMsg:
		if message.request != m.previewRequest || message.appID != m.selectedAppValue().ID {
			return m, nil
		}
		return m, m.loadPreview(message.appID, message.iconURL, message.screenshotURL, message.request)
	case previewLoadedMsg:
		if message.request == m.previewRequest && m.selectedAppValue().ID == message.appID {
			m.iconImage = message.icon
			m.screenshotImage = message.screenshot
		}
	case downloadFinishedMsg:
		m.downloading = false
		if message.err != nil {
			m.status = "下载失败：" + message.err.Error()
			return m, nil
		}
		m.pendingPackage = message.task.Destination
		m.status = "下载完成，正在安装…"
		return m, m.installPackage()
	case uninstallFinishedMsg:
		if message.err != nil {
			m.status = "卸载失败：" + message.err.Error()
		} else {
			m.status = "卸载完成"
		}
		m.pendingPackage = ""
	case installFinishedMsg:
		if message.err != nil {
			m.status = "安装失败：" + message.err.Error()
		} else if m.pendingPackage == "" {
			m.status = "安装完成"
		} else {
			m.status = "安装完成；安装包保留在 " + m.pendingPackage
		}
	case packageStatusMsg:
		if message.appID != m.selectedAppValue().ID {
			return m, nil
		}
		m.checkingPackage = false
		m.packageStatusAppID = message.appID
		if message.err != nil {
			m.packageStatusAppID = ""
			m.status = "检查应用更新失败：" + message.err.Error()
			return m, nil
		}
		m.packageStatus = message.status
		if !message.status.Installed {
			m.status = "当前应用尚未安装；按 D 下载并安装"
		} else if message.status.UpdateAvailable {
			m.status = fmt.Sprintf("发现更新 %s → %s：按 P 升级", message.status.InstalledVersion, message.status.CandidateVersion)
		} else {
			m.status = "当前应用已是软件源中的最新版本：" + message.status.InstalledVersion
		}
	case updateFinishedMsg:
		if message.err != nil {
			m.status = "应用更新失败：" + message.err.Error()
		} else {
			m.packageStatus.Installed = true
			m.packageStatus.InstalledVersion = m.packageStatus.CandidateVersion
			m.packageStatus.UpdateAvailable = false
			m.status = "应用更新完成"
		}
	case previewFinishedMsg:
		if message.err != nil {
			m.status = "预览失败：" + message.err.Error()
		} else {
			m.status = "已返回目录"
		}
	case tea.KeyPressMsg:
		return m.handleKey(message.String())
	}
	return m, nil
}

const helpText = "↑↓/jk 选择 · / 搜索 · [ ] 分类 · D 下载/安装 · I 安装本地包 · P 检查/升级 · U 卸载 · X 删除本地包 · O 原图 · E 简介 · R 刷新 · q 退出"

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	if key == "q" || key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.searching {
		switch key {
		case "esc", "enter":
			m.searching = false
		case "backspace", "ctrl+h":
			m.query = trimLastRune(m.query)
			m.selectedApp = 0
			m.clearPackageStatus()
		default:
			if key == "space" {
				key = " "
			}
			if utf8.RuneCountInString(key) == 1 {
				m.query += key
				m.selectedApp = 0
				m.clearPackageStatus()
			}
		}
		m.iconImage = ""
		m.screenshotImage = ""
		m.previewRequest++
		return m, m.schedulePreview()
	}

	switch key {
	case "esc":
		m.query = ""
		m.focus = focusApps
		m.status = "已清除筛选"
	case "tab", "right", "left":
		if len(m.sources) > 1 {
			if m.focus == focusSources {
				m.focus = focusApps
			} else {
				m.focus = focusSources
			}
		}
	case "up", "k", "down", "j":
		delta := -1
		if key == "down" || key == "j" {
			delta = 1
		}
		if m.focus == focusSources && len(m.sources) > 1 {
			m.selectedSource = wrap(m.selectedSource+delta, len(m.sources))
			m.selectedApp = 0
			m.apps = nil
			m.iconImage = ""
			m.screenshotImage = ""
			m.descriptionExpanded = false
			m.clearPackageStatus()
			m.previewRequest++
			m.loading = true
			m.loadError = ""
			m.categories = nil
			m.status = "正在切换目录源…"
			return m, m.loadCategories()
		}
		if len(m.filteredApps()) > 0 {
			m.selectedApp = wrap(m.selectedApp+delta, len(m.filteredApps()))
			m.iconImage = ""
			m.screenshotImage = ""
			m.descriptionExpanded = false
			m.clearPackageStatus()
			m.previewRequest++
			return m, m.schedulePreview()
		}
	case "/":
		m.searching = true
		m.status = "输入关键词过滤应用；Enter 确认，Esc 取消"
	case "r":
		m.loading = true
		m.loadError = ""
		m.status = "正在刷新目录…"
		m.clearPackageStatus()
		return m, m.loadApps()
	case "[", "]":
		if len(m.categories) == 0 {
			m.status = "当前目录源不提供分类"
			return m, nil
		}
		delta := -1
		if key == "]" {
			delta = 1
		}
		m.categoryIndex = wrap(m.categoryIndex+delta, len(m.categories))
		m.category = m.categories[m.categoryIndex].ID
		m.apps = nil
		m.selectedApp = 0
		m.focus = focusApps
		m.iconImage = ""
		m.screenshotImage = ""
		m.descriptionExpanded = false
		m.clearPackageStatus()
		m.previewRequest++
		m.loading = true
		m.status = "正在切换分类…"
		return m, m.loadApps()
	case "enter":
		if app := m.selectedAppValue(); app.ID == "" {
			m.status = "当前没有可操作的应用"
		} else {
			m.status = fmt.Sprintf("已选择 %s；D 下载/安装 · P 检查更新 · U 卸载", app.Name)
		}
	case "d":
		return m.downloadAction()
	case "i":
		return m.installAction()
	case "p":
		return m.updateAction()
	case "u":
		return m.uninstallAction()
	case "x":
		return m.deleteAction()
	case "o":
		if m.service == nil || m.terminal == nil {
			m.status = "预览服务未配置"
			return m, nil
		}
		command, err := m.service.PreviewCommand(m.selectedAppValue())
		if err != nil {
			m.status = "无法预览：" + err.Error()
			return m, nil
		}
		m.status = "正在打开原图预览…"
		return m, m.terminal.Exec(command, func(err error) tea.Msg { return previewFinishedMsg{err: err} })
	case "e":
		m.descriptionExpanded = !m.descriptionExpanded
	case "?":
		m.status = helpText
	}
	return m, nil
}

func (m Model) queueCatalogRefresh(action string) (tea.Model, tea.Cmd) {
	m.pendingAction = action
	m.pendingAppID = m.selectedAppValue().ID
	m.loading = true
	m.loadError = ""
	m.status = "目录信息超过 10 分钟，正在刷新…"
	return m, m.loadApps()
}

// downloadAction is direct: APM hosts install immediately, and Debian hosts
// install an existing local package instead of opening a confirmation screen.
func (m Model) downloadAction() (tea.Model, tea.Cmd) {
	if m.metadataStale() {
		return m.queueCatalogRefresh("d")
	}
	if m.downloading {
		m.status = "下载任务正在运行"
		return m, nil
	}
	if m.service == nil {
		m.status = "应用管理服务未配置"
		return m, nil
	}
	app := m.selectedAppValue()
	if m.service.UsesAPM() {
		if err := m.service.ValidateAPM(app); err != nil {
			m.status = "当前应用没有可用于 APM 的包名"
			return m, nil
		}
		m.status = "正在通过 Amber APM 安装…"
		return m, m.installPackage()
	}
	if err := m.service.ValidateDownload(app); err != nil {
		m.status = "当前应用没有可交给 aptss 的软件包元数据"
		return m, nil
	}
	if existing := m.service.ExistingPackage(app); existing != "" {
		m.pendingPackage = existing
		m.status = "已存在安装包，正在安装…"
		return m, m.installPackage()
	}
	m.downloading = true
	m.status = "正在通过 aptss/aria2 下载…"
	return m, m.downloadApp()
}

func (m Model) installAction() (tea.Model, tea.Cmd) {
	if m.service == nil || m.terminal == nil {
		m.status = "安装服务未配置"
		return m, nil
	}
	app := m.selectedAppValue()
	path := m.pendingPackage
	if path == "" {
		path = m.service.ExistingPackage(app)
	}
	if path == "" {
		m.status = "没有可安装的本地包；按 D 下载"
		return m, nil
	}
	m.pendingPackage = path
	m.status = "正在安装本地安装包…"
	return m, m.installPackage()
}

func (m Model) updateAction() (tea.Model, tea.Cmd) {
	if m.metadataStale() {
		return m.queueCatalogRefresh("p")
	}
	app := m.selectedAppValue()
	if app.ID == "" {
		m.status = "当前没有可检查更新的应用"
		return m, nil
	}
	if m.packageStatusAppID == app.ID && m.packageStatus.UpdateAvailable {
		plan, err := m.updatePlan()
		if err != nil {
			m.status = "无法更新：" + err.Error()
			return m, nil
		}
		m.status = "正在通过 aptss/apm 更新应用…"
		return m, m.terminal.Exec(plan.Command, func(err error) tea.Msg { return updateFinishedMsg{err: err} })
	}
	if m.checkingPackage {
		m.status = "正在查询所选应用的安装与更新状态"
		return m, nil
	}
	m.checkingPackage = true
	m.status = "正在通过 aptss/apm 检查更新…"
	return m, m.checkPackageStatus()
}

func (m Model) uninstallAction() (tea.Model, tea.Cmd) {
	if m.service == nil || m.terminal == nil {
		m.status = "无法卸载：未配置应用管理服务"
		return m, nil
	}
	if err := m.service.CanUninstall(m.selectedAppValue()); err != nil {
		m.status = "无法卸载：" + err.Error()
		return m, nil
	}
	plan, err := m.uninstallPlan()
	if err != nil {
		m.status = "无法卸载：" + err.Error()
		return m, nil
	}
	m.pendingPackage = m.service.ExistingPackage(m.selectedAppValue())
	m.status = "正在卸载应用…"
	return m, m.terminal.Exec(plan.Command, func(err error) tea.Msg { return uninstallFinishedMsg{err: err} })
}

func (m Model) deleteAction() (tea.Model, tea.Cmd) {
	path := m.pendingPackage
	if path == "" && m.service != nil {
		path = m.service.ExistingPackage(m.selectedAppValue())
	}
	if path == "" {
		m.status = "没有本地安装包可删除"
		return m, nil
	}
	if err := m.deletePackage(path); err != nil {
		m.status = "删除安装包失败：" + err.Error()
	} else {
		m.status = "已删除本地安装包"
	}
	m.pendingPackage = ""
	return m, nil
}

func (m Model) updatePlan() (domain.CommandPlan, error) {
	if m.service == nil || m.terminal == nil {
		return domain.CommandPlan{}, fmt.Errorf("应用管理服务未配置")
	}
	return m.service.Update(m.selectedAppValue())
}

func (m Model) uninstallPlan() (domain.CommandPlan, error) {
	if m.service == nil || m.terminal == nil {
		return domain.CommandPlan{}, fmt.Errorf("应用管理服务未配置")
	}
	return m.service.Uninstall(m.selectedAppValue())
}

func (m Model) deletePackage(path string) error {
	if m.service == nil {
		return fmt.Errorf("应用管理服务未配置")
	}
	return m.service.DeletePackage(path)
}

func (m Model) downloadApp() tea.Cmd {
	app := m.selectedAppValue()
	service := m.service
	if service == nil {
		return func() tea.Msg { return downloadFinishedMsg{err: fmt.Errorf("download service is not configured")} }
	}
	plan, err := service.PrepareDownload(app)
	if err != nil {
		return func() tea.Msg { return downloadFinishedMsg{err: err} }
	}
	if plan.Interactive() {
		if m.terminal == nil {
			return func() tea.Msg { return downloadFinishedMsg{err: fmt.Errorf("terminal is not configured")} }
		}
		return m.terminal.Exec(plan.Command, func(runErr error) tea.Msg {
			task, finishErr := plan.Finish(runErr)
			return downloadFinishedMsg{task: task, err: finishErr}
		})
	}
	if plan.Background == nil {
		return func() tea.Msg { return downloadFinishedMsg{err: fmt.Errorf("download plan is empty")} }
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		task, err := plan.Background(ctx)
		return downloadFinishedMsg{task: task, err: err}
	}
}

func (m Model) checkPackageStatus() tea.Cmd {
	app := m.selectedAppValue()
	service := m.service
	return func() tea.Msg {
		if service == nil {
			return packageStatusMsg{appID: app.ID, err: fmt.Errorf("应用管理服务未配置")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		status, err := service.PackageStatus(ctx, app)
		return packageStatusMsg{appID: app.ID, status: status, err: err}
	}
}

func (m *Model) clearPackageStatus() {
	m.checkingPackage = false
	m.packageStatus = domain.PackageStatus{}
	m.packageStatusAppID = ""
}

func (m Model) installPackage() tea.Cmd {
	path := m.pendingPackage
	if m.service == nil || m.terminal == nil {
		return func() tea.Msg { return installFinishedMsg{err: fmt.Errorf("安装服务未配置")} }
	}
	plan, err := m.service.Install(m.selectedAppValue(), path)
	if err != nil {
		return func() tea.Msg { return installFinishedMsg{err: err} }
	}
	return m.terminal.Exec(plan.Command, func(err error) tea.Msg { return installFinishedMsg{err: err} })
}

func (m Model) loadApps() tea.Cmd {
	source := m.selectedSourceValue()
	service := m.service
	category := m.category
	return func() tea.Msg {
		if service == nil {
			return appsLoadedMsg{sourceID: source.ID, err: fmt.Errorf("%s 尚未配置仓库", source.Name)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		apps, err := service.ListApps(ctx, source.ID, category)
		return appsLoadedMsg{sourceID: source.ID, apps: apps, err: err}
	}
}

func (m Model) loadCategories() tea.Cmd {
	source := m.selectedSourceValue()
	service := m.service
	return func() tea.Msg {
		if service == nil {
			return categoriesLoadedMsg{sourceID: source.ID, err: fmt.Errorf("%s 尚未配置分类或仓库", source.Name)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		categories, err := service.ListCategories(ctx, source.ID)
		return categoriesLoadedMsg{sourceID: source.ID, categories: categories, err: err}
	}
}

// metadataTTL bounds how long a loaded catalog may be used before a package
// action refreshes it, so downloads and upgrades do not rely on stale metadata.
const metadataTTL = 10 * time.Minute

func (m Model) metadataStale() bool {
	return m.loadedAt.IsZero() || time.Since(m.loadedAt) > metadataTTL
}

func (m Model) schedulePreview() tea.Cmd {
	if !m.imageReady || m.width < 80 {
		return nil
	}
	app := m.selectedAppValue()
	iconURL, screenshotURL := m.previewURLs()
	if app.ID == "" || (iconURL == "" && screenshotURL == "") {
		return nil
	}
	return tea.Tick(350*time.Millisecond, func(time.Time) tea.Msg {
		return previewDueMsg{
			appID:         app.ID,
			iconURL:       iconURL,
			screenshotURL: screenshotURL,
			request:       m.previewRequest,
		}
	})
}

// previewURLs returns the metadata icon and the first in-app screenshot. The
// store-style layout shows both; there is no cycling.
func (m Model) previewURLs() (iconURL, screenshotURL string) {
	app := m.selectedAppValue()
	if len(app.ScreenshotURLs) > 0 {
		screenshotURL = app.ScreenshotURLs[0]
	}
	return app.IconURL, screenshotURL
}

func (m Model) loadPreview(appID, iconURL, screenshotURL string, request int) tea.Cmd {
	return func() tea.Msg {
		icon := ""
		if iconURL != "" {
			icon = preview.RenderURLChafa(iconURL, 10, 5)
		}
		screenshot := ""
		if screenshotURL != "" {
			// Bubble Tea's text renderer strips graphics escape sequences, so
			// in-TUI previews use chafa art. The O key opens the original image
			// through the terminal's native protocol outside the TUI.
			screenshot = preview.RenderURLChafa(screenshotURL, 18, 9)
		}
		return previewLoadedMsg{
			appID: appID, icon: icon, screenshot: screenshot, request: request,
		}
	}
}
