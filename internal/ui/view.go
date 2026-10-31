package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/yxxbc/spark-store-tui/internal/domain"
)

func (m Model) View() tea.View { return tea.NewView(m.render()) }

func (m Model) render() string {
	if m.screen == screenHome {
		return m.renderHome()
	}
	return m.renderBrowse()
}

func (m Model) renderHome() string {
	previewStatus := muted.Render("终端图片预览：按 I 查看依赖补给方式")
	if m.imageReady {
		previewStatus = good.Render("终端图片预览：已启用（chafa）")
	}
	intro := strings.Join([]string{
		accent.Render("✦ 星火终端助手"),
		header.Render("在终端中浏览、下载和管理 Linux 软件"),
		"公开 metadata 用于展示；Debian 应用交给 aptss 管理。",
		"aptss 负责搜索软件源、镜像下载、校验、安装与更新。",
		"",
		good.Render("Enter  加载并浏览星火商店"),
		muted.Render("B 浏览应用    ? 帮助    q 退出"),
	}, "\n")
	host := fmt.Sprintf("系统：%s\n发行版族：%s\n架构：%s\nSpark 路径：%s", m.host.Distro, m.host.Family, m.host.Architecture, m.host.SparkArchPath)
	introPanel := panel.Width(max(44, m.width-8)).Render(intro)
	if m.brandImage != "" && m.width >= 120 {
		brandPanel := panel.Copy().Width(24).Height(13).Render(accent.Render("星火标识") + "\n\n" + m.brandImage)
		textPanel := panel.Copy().Width(max(58, m.width-36)).Height(13).Render(intro)
		introPanel = lipgloss.JoinHorizontal(lipgloss.Top, brandPanel, " ", textPanel)
	}
	hostPanel := panel.Width(max(44, m.width-8)).Render(accent.Render("本机自动识别") + "\n" + host + "\n\n" + previewStatus)
	return lipgloss.JoinVertical(lipgloss.Left, introPanel, "", hostPanel, "", muted.Render("启动时不会安装软件；明确执行 --bootstrap-images 后才会调用系统包管理器。"), "", accent.Render(m.status))
}

func (m Model) renderBrowse() string {
	title := header.Render("✦ Spark Store TUI") +
		"  " + muted.Render(m.selectedCategoryName()) +
		"  " + muted.Render(m.selectedSourceValue().Name) +
		"  " + good.Render("● "+m.host.Architecture)
	footer := muted.Render("↑↓ 导航  / 搜索  D 下载  P 更新  U 卸载  E 简介  [ ] 分类  R 刷新  Esc 首页  q 退出")
	if m.width < 80 {
		return m.renderNarrowBrowse()
	}
	bodyHeight := max(8, m.height-6)
	leftWidth := max(38, m.width*42/100)
	rightWidth := max(38, m.width-leftWidth-3)
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderList(leftWidth, bodyHeight), " ", m.renderDetails(rightWidth, bodyHeight, leftWidth+1))
	return lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", footer, accent.Render(m.status))
}

func (m Model) renderNarrowBrowse() string {
	apps := m.filteredApps()
	lines := []string{
		header.Render("✦ Spark Store TUI"),
		muted.Render(m.selectedCategoryName() + " · " + m.host.Architecture),
		m.searchLine(),
		accent.Render(fmt.Sprintf("应用 %d/%d", min(m.selectedApp+1, len(apps)), len(apps))),
	}
	for index, app := range m.windowedApps(apps) {
		actualIndex := m.windowStart(len(apps)) + index
		line := truncate(app.Name+" · "+valueOr(app.Version, "未知"), max(18, m.width-2))
		if actualIndex == m.selectedApp {
			line = selected.Render("› " + line)
		}
		lines = append(lines, line)
	}
	app := m.selectedAppValue()
	if app.ID != "" {
		lines = append(lines, "", accent.Render("详情"), truncate(valueOr(app.Description, "暂无简介"), max(20, m.width-2)))
	}
	lines = append(lines, muted.Render("/ 搜索  D 下载  P 更新  U 卸载  E 简介  [ ] 分类  q 退出"), accent.Render(m.status))
	return strings.Join(lines, "\n")
}

func (m Model) renderList(width, height int) string {
	style := panel.Width(width - 4).Height(max(4, height-4))
	if m.focus == focusApps {
		style = focusedPanel.Width(width - 4).Height(max(4, height-4))
	}
	lines := []string{
		accent.Render("应用列表") + muted.Render("  "+m.selectedCategoryName()),
		m.searchLine(),
	}
	if m.loading {
		lines = append(lines, muted.Render("正在请求公开 metadata…"))
	}
	if m.loadError != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(m.loadError))
	}
	apps := m.filteredApps()
	for index, app := range m.windowedApps(apps) {
		actualIndex := m.windowStart(len(apps)) + index
		line := truncate(app.Name+"  "+valueOr(app.Version, "未知")+" · "+valueOr(app.PackageFormat, "包"), max(12, width-18))
		if actualIndex == m.selectedApp {
			line = selected.Width(max(12, width-16)).Render("› " + line)
		}
		lines = append(lines, line)
	}
	if !m.loading && m.loadError == "" && len(apps) == 0 {
		lines = append(lines, muted.Render("当前目录没有可显示的应用。"))
	}
	if len(apps) > 0 {
		lines = append(lines, muted.Render(fmt.Sprintf("%d / %d", m.selectedApp+1, len(apps))))
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderDetails(width, height, offset int) string {
	app := m.selectedAppValue()
	if height < 24 {
		lines := []string{
			accent.Render("应用详情"),
			header.Render(valueOr(app.Name, "加载中")),
			"版本：" + valueOr(app.Version, "待加载"),
			"包格式：" + valueOr(app.PackageFormat, "待加载"),
			"大小：" + valueOr(app.Size, "待加载"),
			m.packageStatusLine(app),
			"",
			muted.Render("↑↓ 选择 · D 下载 · P 更新 · U 卸载"),
			muted.Render("放大终端以显示简介和截图"),
		}
		return panel.Width(width - 4).Height(max(4, height-4)).Render(strings.Join(lines, "\n"))
	}

	icon := m.iconImage
	if icon == "" {
		icon = muted.Render("图标")
	}
	meta := strings.Join([]string{
		header.Render(valueOr(app.Name, "加载中")),
		"版本：" + valueOr(app.Version, "待加载"),
		"包格式：" + valueOr(app.PackageFormat, "待加载"),
		"大小：" + valueOr(app.Size, "待加载"),
		"来源：" + sourceLabel(app.SourceID),
		m.packageStatusLine(app),
	}, "\n")
	headerBlock := lipgloss.JoinHorizontal(lipgloss.Top, icon, "  ", meta)

	description := descriptionForDisplay(valueOr(app.Description, "真实目录接入后显示完整简介。"), width, m.descriptionExpanded)
	descriptionHint := muted.Render("按 E 展开简介")
	if m.descriptionExpanded {
		descriptionHint = muted.Render("按 E 收起简介")
	}
	downloadHint := muted.Render("D 下载 · P 检查更新 · U 卸载")
	if m.downloading {
		downloadHint = accent.Render("下载任务运行中…")
	}

	screenshot := m.screenshotImage
	if !m.imageReady {
		screenshot = muted.Render("图片预览：安装 chafa 或使用支持图片的终端")
	}
	native := isNativeImage(screenshot)
	lines := []string{
		accent.Render("应用详情"),
		headerBlock,
		"",
		description,
		descriptionHint,
		"",
		muted.Render("界面截图"),
	}
	if !native {
		if screenshot == "" {
			screenshot = muted.Render("暂无界面截图")
		}
		lines = append(lines, screenshot, "")
	}
	lines = append(lines, downloadHint, muted.Render("Debian：aptss/aria2 负责镜像选择、续传和软件源摘要校验"))

	contentHeight := max(4, height-4)
	if native {
		contentHeight = max(4, height-15)
	}
	rendered := panel.Width(width - 4).Height(contentHeight).Render(strings.Join(lines, "\n"))
	if native {
		imageBlock := strings.Repeat(" ", offset) + screenshot + strings.Repeat("\n", 9)
		return lipgloss.JoinVertical(lipgloss.Left, rendered, imageBlock)
	}
	return rendered
}

func isNativeImage(value string) bool {
	return strings.HasPrefix(value, "\x1b_G") || strings.HasPrefix(value, "\x1b]1337;")
}

func (m Model) packageStatusLine(app domain.App) string {
	if m.checkingPackage && m.packageStatusAppID == "" {
		return "安装状态：正在查询…"
	}
	if app.ID == "" || m.packageStatusAppID != app.ID {
		return "安装状态：按 P 检查更新"
	}
	if !m.packageStatus.Installed {
		return "安装状态：未安装"
	}
	if m.packageStatus.UpdateAvailable {
		return fmt.Sprintf("安装状态：%s，可更新至 %s", m.packageStatus.InstalledVersion, m.packageStatus.CandidateVersion)
	}
	return "安装状态：已是最新 " + m.packageStatus.InstalledVersion
}

func (m Model) selectedAppValue() domain.App {
	apps := m.filteredApps()
	if len(apps) == 0 {
		return domain.App{}
	}
	index := m.selectedApp
	if index >= len(apps) {
		index = 0
	}
	return apps[index]
}

func (m Model) filteredApps() []domain.App {
	if m.query == "" {
		return m.apps
	}
	query := strings.ToLower(m.query)
	apps := make([]domain.App, 0, len(m.apps))
	for _, app := range m.apps {
		content := strings.ToLower(strings.Join([]string{app.Name, app.Description, app.Version, app.PackageFormat}, " "))
		if strings.Contains(content, query) {
			apps = append(apps, app)
		}
	}
	return apps
}

func (m Model) searchLine() string {
	if m.searching {
		return accent.Render("搜索：") + m.query + "_"
	}
	if m.query != "" {
		return accent.Render("筛选：") + m.query + muted.Render("  （按 / 修改）")
	}
	return muted.Render("按 / 搜索应用")
}

func (m Model) windowedApps(apps []domain.App) []domain.App {
	if len(apps) == 0 {
		return nil
	}
	start := m.windowStart(len(apps))
	end := min(start+m.visibleRows(), len(apps))
	return apps[start:end]
}

func (m Model) windowStart(count int) int {
	rows := m.visibleRows()
	if count <= rows {
		return 0
	}
	start := m.selectedApp - rows/2
	if start < 0 {
		return 0
	}
	if start+rows > count {
		return count - rows
	}
	return start
}

func (m Model) visibleRows() int {
	return max(4, min(24, m.height-14))
}

func wrap(value, count int) int {
	value %= count
	if value < 0 {
		value += count
	}
	return value
}

func trimLastRune(value string) string {
	if value == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}

func truncate(value string, limit int) string {
	if limit <= 1 {
		return value
	}
	return ansi.Truncate(value, limit, "…")
}

func descriptionForDisplay(value string, width int, expanded bool) string {
	value = strings.Join(strings.Fields(value), " ")
	if expanded {
		return truncate(value, 1600)
	}
	limit := max(90, min(260, width*4))
	if runewidth.StringWidth(value) <= limit {
		return value
	}
	return truncate(value, limit) + " " + muted.Render("…")
}

func (m Model) selectedSourceValue() domain.CatalogSource {
	if len(m.sources) == 0 || m.selectedSource >= len(m.sources) {
		return domain.CatalogSource{}
	}
	return m.sources[m.selectedSource]
}

func (m Model) selectedCategoryName() string {
	if len(m.categories) > 0 && m.categoryIndex < len(m.categories) {
		return m.categories[m.categoryIndex].Name
	}
	return m.category
}

func categoryIndex(categories []domain.Category, id string) int {
	for index, category := range categories {
		if category.ID == id {
			return index
		}
	}
	return 0
}

func sourceLabel(id string) string {
	if id == "" {
		return "-"
	}
	return "Spark"
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
