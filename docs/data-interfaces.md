# 数据接口与内部边界（0.9.1）

本文是 `sparkstore` 的数据契约文档。代码中的接口均定义在消费者包内，便于替换实现和单元测试；外部程序不要依赖 `internal/` 包路径，稳定契约以本文为准。

## 分层与依赖方向

```text
domain, version, config, preview        ← 叶子层，仅依赖标准库/渲染库
platform, catalog, download, state      → domain
packages                                → domain, platform
app                                     → domain, config, platform, catalog, download, packages
ui                                      → domain, preview（仅此二者）
cmd/sparkstore                          → app, ui, config, platform, version
```

`internal/ui` 不导入 `catalog`、`packages`、`platform`、`download`、`state`，也不导入 `os/exec`。所有进程交接通过 `ui.Terminal` 端口完成。

## 领域模型（`internal/domain`）

```go
type App struct {
    ID, Name, Version, PackageFormat, Size, Description string
    SourceID, Architecture, Checksum, Category string
    PackageName, Filename, MetalinkURL, IconURL string
    ScreenshotURLs []string
    Assets []ReleaseAsset
}

type CatalogSource struct {
    ID, Name string
    Kind SourceKind
    Description string
    Mirrors []Mirror
}

type Category struct{ ID, Name string }

type DownloadTask struct {
    ID, AppID, SourceID, MirrorID, URL, Destination string
    ExpectedSHA256, ExpectedSHA1 string
    BytesCompleted, BytesTotal int64
    Status TaskStatus
    Error string
    CreatedAt, UpdatedAt time.Time
}

type PackageStatus struct {
    PackageName, InstalledVersion, CandidateVersion string
    Installed, UpdateAvailable bool
}

type PlatformInfo struct {
    Distro, Family, Architecture, SparkArchPath string
}
```

### 图标与截图

- `App.IconURL` 来自 metadata 的 `icons`，表示应用图标。
- `App.ScreenshotURLs` 来自 metadata 的 `img_urls`，表示应用内界面截图。
- 详情页采用标准商店排版：应用图标 + 首张截图同时显示，不做切换；没有截图时只显示图标。

### 操作计划

UI 不构造命令，只消费计划：

```go
type Command struct {
    Path string
    Args []string // 不含 argv[0]
    Dir  string
    Env  []string // nil 表示继承
}

type DownloadPlan struct {
    Command    Command
    Finish     func(runErr error) (DownloadTask, error)
    Background func(context.Context) (DownloadTask, error)
}

type CommandPlan struct{ Command Command }
```

- `DownloadPlan.Command.Path != ""`：交互式下载（aptss/aria2），由 `ui.Terminal` 接管终端。
- 否则使用 `Background`：HTTP/Metalink 下载在 TUI 进程内完成。
- `CommandPlan` 只包含已经过校验的命令，UI 不拼接参数。

## 目录源（`internal/catalog`）

```go
type Query struct {
    Architecture string
    Category     string
    Limit        int
}

type Provider interface {
    Source() CatalogSource
    ListApps(context.Context, Query) ([]App, error)
}

type CategoryProvider interface {
    ListCategories(context.Context) ([]Category, error)
}
```

实现：`SparkMetadataProvider`。它只读取公开 metadata，不执行外部进程。

## 下载层（`internal/download`）

```go
type TaskStore interface {
    Upsert(DownloadTask) error
    Load() ([]DownloadTask, error)
}

// MetalinkDownloader
func (d *MetalinkDownloader) Prepare(App) (domain.DownloadPlan, error)
func (d *MetalinkDownloader) Download(context.Context, App) (domain.DownloadTask, error)
func (d *MetalinkDownloader) ExistingPath(App) string
func (d *MetalinkDownloader) Recover() ([]domain.DownloadTask, error)

// APTSSDownloader
func (d *APTSSDownloader) Prepare(App) (domain.DownloadPlan, error)
func (d *APTSSDownloader) ExistingPath(App) string
func (d *APTSSDownloader) Recover() ([]domain.DownloadTask, error)
```

`app.Downloader` 只要求 `Prepare`、`ExistingPath`、`Recover`，所以测试可以注入假实现。

### 数据校验规则

- 文件名必须是单段文件名（拒绝 `/`、`\`、`..`、控制字符）。
- Metalink URL 只接受 `http(s)`，拒绝内嵌凭据；默认拒绝明文 HTTP。
- 响应体有上限：Metalink 1 MiB，图片 12 MiB。
- 下载先写 `.part`，校验摘要后原子 rename；中断任务持久化到 `tasks.json`。
- 摘要优先级：SHA-256 > SHA-1；都没有时仅依赖 HTTPS。

## 包后端（`internal/packages`）

```go
type Backend interface {
    Family() string
    Validate(App) error
    Install(App, localPath string) (Command, error)
    Remove(App) (Command, error)
    Update(App) (Command, error)
    Status(context.Context, App) (PackageStatus, error)
}

func For(platform.Host) (Backend, error)
```

- `deb` → `APTSS`
- `arch`、`rpm`、`suse` → `APM`
- 命令通过 `exec.Command` 参数列表构造，不经过 shell；非 root 时统一使用 `sudo -- <cmd>`。
- 包名解析优先取 Deb 文件名前缀，并做字符白名单校验。
- WSL 下清理 PATH 中的 Windows 路径，避免 APM 的 bwrap/shell 误解析。

## 应用服务（`internal/app`）

```go
type Service struct{ ... }
func New(Deps) *Service

func (s *Service) Sources() []CatalogSource
func (s *Service) Host() domain.PlatformInfo
func (s *Service) ListCategories(ctx, sourceID) ([]Category, error)
func (s *Service) ListApps(ctx, sourceID, category) ([]App, error)
func (s *Service) UsesAPM() bool
func (s *Service) ValidateAPM(App) error
func (s *Service) ValidateDownload(App) error
func (s *Service) ExistingPackage(App) string
func (s *Service) PrepareDownload(App) (DownloadPlan, error)
func (s *Service) Install(App, localPath) (CommandPlan, error)
func (s *Service) Update(App) (CommandPlan, error)
func (s *Service) Uninstall(App) (CommandPlan, error)
func (s *Service) CanUninstall(App) error
func (s *Service) DeletePackage(path) error // 仅限下载目录
func (s *Service) PackageStatus(ctx, App) (PackageStatus, error)
func (s *Service) ImagePreviewAvailable() bool
func (s *Service) BootstrapCommand() (string, bool)
func (s *Service) Recover() ([]DownloadTask, error)
```

`Deps` 显式注入 `Sources`、`Catalogs`、`Backend`、`Downloader`、`Host`、`Paths`，不存在包级全局状态。

## UI 端口（`internal/ui`）

```go
type Service interface { /* 上述 app.Service 方法的 UI 子集 */ }

type Terminal interface {
    Exec(domain.Command, func(error) tea.Msg) tea.Cmd
}
```

`cmd/sparkstore/terminal.go` 是唯一把 `domain.Command` 转成 `*exec.Cmd` 并调用 `tea.ExecProcess` 的位置。

## 持久化（`internal/state`）

任务快照保存在 `SPARKSTORE_CONFIG_DIR`（默认 `~/.config/sparkstore/tasks.json`），写临时文件后 rename，进程崩溃不会留下半截 JSON。`0.8.x` 的路径保持不变，升级后可继续恢复中断任务。

## 元数据新鲜度

- `catalog` 不使用离线 applist 缓存，请求带 `Cache-Control: no-cache`。
- `app.Service` 在 `D` / `P` 前检查目录加载时间；超过 10 分钟先刷新目录，再执行操作。
- applist 按包名去重，带路径分隔符的文件名被丢弃。
- 图片缓存 24 小时后重新校验。

## 机器可读 schema

- [`schema/applist.schema.json`](schema/applist.schema.json)
- [`schema/categories.schema.json`](schema/categories.schema.json)
- [`schema/download-task.schema.json`](schema/download-task.schema.json)

## 扩展新的数据源

1. 在 `internal/catalog` 实现 `Provider`（按需实现 `CategoryProvider`）。
2. 在 `cmd/sparkstore/main.go` 注册到 `Catalogs` 映射。
3. 若是包格式/后端差异，在 `internal/packages` 新增 `Backend` 并在 `For` 中映射。
4. 不要从 UI 直接调用新后端；保持 `app.Service` 为唯一编排点。
