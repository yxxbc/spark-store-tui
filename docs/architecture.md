# 架构（0.9.1）

0.9.1 保持用户可见行为不变，重写了内部控制层。目标是把“目录数据、下载、包管理、UI”拆成可替换的边界，并让所有提权操作只有一个出口。

## 目录

```text
cmd/sparkstore/          程序入口、flag、依赖装配、Terminal 适配器
internal/
  app/                   用例编排：目录、下载、安装、更新、卸载、状态
  catalog/               Spark metadata Provider（原 provider）
  config/                XDG 路径与 SPARKSTORE_* 环境变量
  domain/                纯模型：App、CatalogSource、DownloadTask、Plan、Status
  download/              Metalink/HTTP 引擎、aptss 下载器、任务生命周期
  packages/              aptss / Amber APM 后端、包名解析、policy 解析
  platform/              发行版/架构探测、图片预览依赖
  preview/               原图协议与 chafa 兜底
  state/                 原子 JSON 任务快照
  ui/                    Bubble Tea 模型/视图；只依赖 domain 与 preview
  version/               ldflags 注入的版本信息
packaging/
  debian/package-root/   Debian 包根、.desktop、图标
  rpm/                   RPM spec
  aur/                   PKGBUILD/SRCINFO 模板
assets/                  icon.svg / icon-official.svg 及渲染图
AGENTS.md                贡献与代理约定
CHANGELOG.md             版本变更与发布说明来源
scripts/                 build-dist、sync-all、install、publish-aur
.github/workflows/       ci、release、mirror、publish-aur
.cnb.yml                 CNB CI 与 tag 发布
VERSION                  唯一上游版本源
```

## 依赖方向

```text
domain, version, config, preview         叶子
platform, catalog, download, state      → domain
packages                                → domain, platform
app                                     → domain, config, platform, catalog, download, packages
ui                                      → domain, preview
cmd/sparkstore                          → app, ui, config, platform, version
```

`go list -deps ./internal/ui` 只应出现 `internal/domain` 与 `internal/preview`。UI 不导入 `os/exec`，也不直接删除文件。

## 关键接口

- `catalog.Provider` / `catalog.CategoryProvider`：上游目录数据的唯一入口。
- `app.Downloader`：`Prepare` / `ExistingPath` / `Recover`。
- `packages.Backend`：`Validate` / `Install` / `Remove` / `Update` / `Status`；`packages.For` 按发行版族选择 aptss 或 APM。
- `ui.Service` / `ui.Terminal`：UI 的消费者侧端口；`cmd/sparkstore/terminal.go` 是唯一构造 `exec.Cmd` 的地方。

完整签名与数据契约见 [data-interfaces.md](data-interfaces.md)。

## 下载生命周期

1. `app.Service.PrepareDownload` 选择下载器。
2. Debian：`APTSSDownloader` 生成 `domain.Command`，通过 `Terminal` 交给 `tea.ExecProcess`，aptss/aria2 显示原生进度并负责续传。
3. 非 Debian：`MetalinkDownloader` 在后台解析 Metalink，按 preference 与 HTTPS 优先排序镜像，逐个尝试。
4. `Engine` 写 `.part`、支持 Range 续传、无数据超时、摘要校验、原子 rename。
5. 任务状态写入 `~/.config/sparkstore/tasks.json`；启动时 `Recover` 结合最终文件、`.aria2` 与 `.part` 判断完成/中断。

## 包管理

- Debian：`aptss download/install/remove/policy`。
- Arch/RPM/openSUSE：`apm install/remove/policy/install --only-upgrade`。
- 包名优先取 Deb 文件名前缀；所有注入路径走参数列表与白名单。
- 非 root 使用 `sudo --`，root 直接执行。

详见 [package-backends.md](package-backends.md)。

## 图片预览

`internal/preview` 先检测终端原生图片协议：

- TUI 内：统一用 `chafa --format=symbols` 输出文本图，因为 Bubble Tea v2 的文本渲染器会丢弃终端图形转义序列。
- `O` 键：TUI 暂停后由子进程输出原图；Kitty / Ghostty 使用 Kitty graphics protocol，iTerm2 / WezTerm 使用 inline image protocol，其他终端回退 chafa。

远程图片限制 12 MiB、要求 `image/*`，缓存目录 `0700`、文件 `0600`。

## 配置路径

| 变量 | 默认值 |
|---|---|
| `SPARKSTORE_CONFIG_DIR` | `~/.config/sparkstore` |
| `SPARKSTORE_DOWNLOAD_DIR` | `~/Downloads` |
| `SPARKSTORE_ALLOW_HTTP` | 未设置（只允许 HTTPS） |
| `SPARKSTORE_BRAND_IMAGE` | 工作目录 `icon.png` |

## 版本与构建

`VERSION` 是唯一上游版本；`packaging/revisions.env` 保存 Deb/RPM/source/AUR 修订号。`scripts/build-dist.sh` 用 `-ldflags` 注入 `version.Version/Commit/Date`，并生成所有发布物与 `SHA256SUMS`。`scripts/check-version.sh` 校验 spec、control、AUR 模板与旧路径残留。

## 仓库角色

- **Gitee**：主线仓库与国内用户入口。
- **GitHub**：构建与发布包、Go module 路径、海外用户镜像。
- **CNB**：镜像与 CNB 原生流水线。

## 发布

- GitHub Actions：`ci.yml`（测试/交叉构建/dist 冒烟）、`release.yml`（tag 构建并发布 GitHub，随后用 repo-all-sync 发布 Gitee/CNB）、`mirror.yml`（main/tag 源码镜像）。
- CNB：`.cnb.yml` 在 main 跑测试与交叉构建，在 `v*` tag 构建产物并发布 CNB/Gitee。
- 本地：`scripts/sync-all.sh` 串联 `doctor → push → release`，`--dry-run` 可预演。

## 开发

```bash
go test ./...
go vet ./...
make go-build
./build/sparkstore --version
```
