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

type screen int

const (
	screenHome screen = iota
	screenBrowse
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
}

// Terminal owns process handover so the UI never imports os/exec.
type Terminal interface {
	Exec(domain.Command, func(error) tea.Msg) tea.Cmd
}

// Model renders the catalog and drives confirmed package operations.
type Model struct {
	service              Service
	terminal             Terminal
	sources              []domain.CatalogSource
	apps                 []domain.App
	host                 domain.PlatformInfo
	screen               screen
	focus                focusPane
	selectedSource       int
	selectedApp          int
	category             string
	categories           []domain.Category
	categoryIndex        int
	width                int
	height               int
	status               string
	loadError            string
	loading              bool
	imageReady           bool
	brandImage           string
	previewImage         string
	previewRequest       int
	searching            bool
	query                string
	descriptionExpanded  bool
	pendingUninstall     bool
	pendingExisting      bool
	pendingInstall       bool
	pendingPackage       string
	deleteAfterUninstall bool
	downloading          bool
	checkingPackage      bool
	pendingUpdate        bool
	packageStatus        domain.PackageStatus
	packageStatusAppID   string
}

func New(service Service, terminal Terminal) Model {
	model := Model{
		service:    service,
		terminal:   terminal,
		category:   "development",
		width:      120,
		height:     38,
		status:     "按 Enter 加载真实星火商店目录",
		brandImage: preview.Brand(),
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

func (m Model) Init() tea.Cmd { return nil }

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
		m.clearPackageStatus()
		m.previewRequest++
		if message.err != nil {
			m.loadError = message.err.Error()
			m.status = "目录加载失败"
			return m, nil
		}
		m.loadError = ""
		m.status = fmt.Sprintf("已加载 %d 个真实应用（%s）", len(m.apps), m.category)
		return m, m.schedulePreview()
	case previewDueMsg:
		if message.request != m.previewRequest || message.appID != m.selectedAppValue().ID {
			return m, nil
		}
		return m, m.loadPreview(message.appID, message.url, message.request)
	case previewLoadedMsg:
		if message.request == m.previewRequest && m.selectedAppValue().ID == message.appID {
			m.previewImage = message.image
		}
	case downloadFinishedMsg:
		m.downloading = false
		if message.err != nil {
			m.status = "下载失败：" + message.err.Error()
		} else {
			m.pendingPackage = message.task.Destination
			m.pendingInstall = true
			m.status = "下载完成：I 或 Enter 安装；Esc 保留安装包"
		}
	case uninstallFinishedMsg:
		if message.err != nil {
			m.status = "卸载失败：" + message.err.Error()
		} else {
			m.status = "卸载完成"
			if m.deleteAfterUninstall && m.pendingPackage != "" && m.service != nil {
				if err := m.service.DeletePackage(m.pendingPackage); err != nil {
					m.status = "已卸载应用，但删除安装包失败：" + err.Error()
				} else {
					m.status = "已卸载应用并删除安装包"
				}
			}
		}
		m.deleteAfterUninstall = false
		m.pendingPackage = ""
	case installFinishedMsg:
		m.pendingInstall = false
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
			m.pendingUpdate = true
			m.status = fmt.Sprintf("发现更新 %s → %s：P 或 Enter 升级；Esc 取消", message.status.InstalledVersion, message.status.CandidateVersion)
		} else {
			m.status = "当前应用已是软件源中的最新版本：" + message.status.InstalledVersion
		}
	case updateFinishedMsg:
		m.pendingUpdate = false
		if message.err != nil {
			m.status = "应用更新失败：" + message.err.Error()
		} else {
			m.packageStatus.Installed = true
			m.packageStatus.InstalledVersion = m.packageStatus.CandidateVersion
			m.packageStatus.UpdateAvailable = false
			m.status = "应用更新完成"
		}
	case tea.KeyPressMsg:
		return m.handleKey(message.String())
	}
	return m, nil
}

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	if key == "q" || key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.searching {
		switch key {
		case "esc":
			m.searching = false
		case "enter":
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
		m.previewImage = ""
		m.previewRequest++
		return m, m.schedulePreview()
	}
	if m.pendingUpdate {
		switch key {
		case "p", "enter":
			plan, err := m.updatePlan()
			m.pendingUpdate = false
			if err != nil {
				m.status = "无法更新：" + err.Error()
				return m, nil
			}
			m.status = "正在通过 aptss/apm 更新应用…"
			return m, m.terminal.Exec(plan.Command, func(err error) tea.Msg { return updateFinishedMsg{err: err} })
		case "esc":
			m.pendingUpdate = false
			m.status = "已取消应用更新"
			return m, nil
		default:
			m.status = "发现可用更新：P 或 Enter 升级；Esc 取消"
			return m, nil
		}
	}
	if m.pendingUninstall {
		switch key {
		case "u", "enter", "a":
			plan, err := m.uninstallPlan()
			m.pendingUninstall = false
			if err != nil {
				m.status = "无法卸载：" + err.Error()
				return m, nil
			}
			m.status = "正在执行卸载…"
			m.deleteAfterUninstall = key == "a"
			return m, m.terminal.Exec(plan.Command, func(err error) tea.Msg { return uninstallFinishedMsg{err: err} })
		case "x":
			m.pendingUninstall = false
			if m.pendingPackage == "" {
				m.status = "没有本地安装包可删除"
				return m, nil
			}
			if err := m.deletePackage(m.pendingPackage); err != nil {
				m.status = "删除安装包失败：" + err.Error()
			} else {
				m.status = "已删除安装包"
			}
			m.pendingPackage = ""
			return m, nil
		case "esc":
			m.pendingUninstall = false
			m.status = "已取消卸载"
			return m, nil
		default:
			m.status = "再按 U 或 Enter 确认卸载；Esc 取消"
			return m, nil
		}
	}
	if m.pendingExisting {
		switch key {
		case "i", "enter":
			m.pendingExisting = false
			m.pendingInstall = true
			return m, m.installPackage()
		case "d":
			m.pendingExisting = false
			m.status = "正在重新下载…"
			m.downloading = true
			return m, m.downloadApp()
		case "x":
			m.pendingExisting = false
			if err := m.deletePackage(m.pendingPackage); err != nil {
				m.status = "删除安装包失败：" + err.Error()
			} else {
				m.status = "已删除安装包，按 D 重新下载"
			}
			m.pendingPackage = ""
			return m, nil
		case "esc":
			m.pendingExisting = false
			m.status = "已取消操作，保留安装包"
			return m, nil
		default:
			m.status = "已存在安装包：I 安装 · D 重新下载 · X 删除 · Esc 取消"
			return m, nil
		}
	}
	if m.pendingInstall {
		switch key {
		case "i", "enter":
			return m, m.installPackage()
		case "esc":
			m.pendingInstall = false
			m.status = "已保留安装包，可稍后按 D 再选择安装"
			return m, nil
		default:
			m.status = "下载完成：I 或 Enter 安装；Esc 保留安装包"
			return m, nil
		}
	}
	if m.screen == screenHome {
		switch key {
		case "enter", "b":
			m.screen = screenBrowse
			m.status = "正在加载星火商店分类…"
			m.loading = true
			return m, m.loadCategories()
		case "?":
			m.status = "Enter/B 浏览应用 · q 退出 · 浏览页中按 Esc 返回简介"
		}
		return m, nil
	}

	switch key {
	case "esc", "h":
		m.screen = screenHome
		m.status = "已返回星火商店简介"
	case "tab", "right", "left":
		if m.focus == focusSources {
			m.focus = focusApps
		} else {
			m.focus = focusSources
		}
	case "up", "k", "down", "j":
		delta := -1
		if key == "down" || key == "j" {
			delta = 1
		}
		if m.focus == focusSources && len(m.sources) > 0 {
			m.selectedSource = wrap(m.selectedSource+delta, len(m.sources))
			m.selectedApp = 0
			m.apps = nil
			m.previewImage = ""
			m.descriptionExpanded = false
			m.clearPackageStatus()
			m.previewRequest++
			m.loading = true
			m.loadError = ""
			m.categories = nil
			m.status = "正在切换目录源…"
			return m, m.loadCategories()
		}
		if m.focus == focusApps && len(m.filteredApps()) > 0 {
			m.selectedApp = wrap(m.selectedApp+delta, len(m.filteredApps()))
			m.previewImage = ""
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
		m.previewImage = ""
		m.descriptionExpanded = false
		m.clearPackageStatus()
		m.previewRequest++
		m.loading = true
		m.status = "正在切换分类…"
		return m, m.loadApps()
	case "enter":
		app := m.selectedAppValue()
		if app.ID == "" {
			m.status = "当前没有可下载应用"
		} else {
			m.status = fmt.Sprintf("已选择 %s；下载任务接入完成后将从详情资产创建", app.Name)
		}
	case "d":
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
			m.pendingInstall = true
			m.status = "将通过 Amber APM 安装：I 或 Enter 确认；Esc 取消"
			return m, nil
		}
		if err := m.service.ValidateDownload(app); err != nil {
			m.status = "当前应用没有可交给 aptss 的软件包元数据"
			return m, nil
		}
		if existing := m.service.ExistingPackage(app); existing != "" {
			m.pendingExisting = true
			m.pendingPackage = existing
			m.status = "已存在安装包：I 安装 · D 重新下载 · X 删除 · Esc 取消"
			return m, nil
		}
		m.downloading = true
		m.status = "正在通过 aptss/aria2 下载或续传…"
		return m, m.downloadApp()
	case "p":
		if m.checkingPackage {
			m.status = "正在查询所选应用的安装与更新状态"
			return m, nil
		}
		if m.selectedAppValue().ID == "" {
			m.status = "当前没有可检查更新的应用"
			return m, nil
		}
		m.checkingPackage = true
		m.status = "正在通过 aptss/apm 检查所选应用更新…"
		return m, m.checkPackageStatus()
	case "u":
		if m.service == nil {
			m.status = "无法卸载：未配置应用管理服务"
			return m, nil
		}
		if err := m.service.CanUninstall(m.selectedAppValue()); err != nil {
			m.status = "无法卸载：" + err.Error()
			return m, nil
		}
		m.pendingUninstall = true
		m.pendingPackage = m.service.ExistingPackage(m.selectedAppValue())
		m.status = "U/Enter 卸载应用 · X 删除安装包 · A 两者都删除 · Esc 取消"
	case "e":
		m.descriptionExpanded = !m.descriptionExpanded
	case "i":
		if m.imageReady {
			m.status = "图片预览已启用；使用应用图标，截图会在详情扩展后显示"
		} else if m.service != nil {
			if command, ok := m.service.BootstrapCommand(); ok {
				m.status = "执行 --bootstrap-images 安装图片预览依赖：" + command
				return m, nil
			}
			m.status = "当前发行版尚未配置图片预览依赖；请安装 chafa"
		} else {
			m.status = "当前发行版尚未配置图片预览依赖；请安装 chafa"
		}
	case "?":
		m.status = "↑↓ 选择 · / 搜索 · D 下载 · P 更新 · U 卸载 · E 简介 · [ ] 分类 · R 刷新 · q 退出"
	}
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
	m.pendingUpdate = false
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

func (m Model) schedulePreview() tea.Cmd {
	if !m.imageReady || m.width < 80 {
		return nil
	}
	app := m.selectedAppValue()
	url := app.IconURL
	if url == "" && len(app.ScreenshotURLs) > 0 {
		url = app.ScreenshotURLs[0]
	}
	if app.ID == "" || url == "" {
		return nil
	}
	return tea.Tick(350*time.Millisecond, func(time.Time) tea.Msg {
		return previewDueMsg{appID: app.ID, url: url, request: m.previewRequest}
	})
}

func (m Model) loadPreview(appID, url string, request int) tea.Cmd {
	return func() tea.Msg {
		return previewLoadedMsg{appID: appID, image: preview.RenderURL(url), request: request}
	}
}
