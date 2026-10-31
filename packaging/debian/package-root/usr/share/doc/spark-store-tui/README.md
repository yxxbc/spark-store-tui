<p align="center">
  <img src="icon.png" width="180" alt="Spark Store TUI logo">
</p>

<h1 align="center">Spark Store TUI</h1>

<p align="center">星火终端助手 · 原生 Linux 软件管理终端界面</p>

<p align="center">
  <a href="https://gitee.com/spark-store-project/spark-store-tui"><img src="https://img.shields.io/badge/Release-v0.9.0-ff9d2e" alt="Release v0.9.0"></a>
  <a href="https://gitee.com/spark-store-project/spark-store-tui"><img src="https://img.shields.io/badge/Gitee-Source-C71D23?logo=gitee&logoColor=white" alt="Gitee source"></a>
  <a href="https://github.com/yxxbc/spark-store-tui"><img src="https://img.shields.io/badge/GitHub-Mirror-181717?logo=github&logoColor=white" alt="GitHub mirror"></a>
  <a href="https://cnb.cool/yxxbc/spark-store-tui"><img src="https://img.shields.io/badge/CNB-Mirror-1f6feb" alt="CNB mirror"></a>
  <a href="COPYING"><img src="https://img.shields.io/badge/License-GPL--3.0--only-4caf50" alt="GPL-3.0-only"></a>
  <img src="https://img.shields.io/badge/Go-%3E%3D1.25-00ADD8?logo=go&logoColor=white" alt="Go 1.25 or newer">
</p>

> `sparkstore` 使用 Spark Store 公开 metadata 展示目录；Debian 应用交给官方 `aptss` 下载、校验、安装、更新和卸载，其他受支持发行版交给 Amber APM。

当前稳定版：**0.9.0**；Debian 包修订版：**0.9.0-1**。

## 0.9.0 主要变化

- **底层重写**：新增 `internal/app` 用例层、`internal/packages` 包后端抽象、`internal/config`、`internal/version`；UI 只依赖 `domain` 与 `preview`，不再直接拼 `exec.Cmd` 或删除文件。
- **数据契约文档化**：[数据接口](docs/data-interfaces.md) 与 [包管理链路调研](docs/package-backends.md)。
- **安全加固**：命令全部走参数列表、`sudo --`、包名/文件名白名单、下载目录边界、Metalink 大小与 URL 校验、SHA-256 优先/SHA-1 兼容校验；详见 [SECURITY.md](SECURITY.md)。
- **原图直出 + chafa 兜底**：Kitty / iTerm2 / WezTerm / Ghostty 走原生图片协议，其余终端回退 `chafa`。
- **GUI 桌面入口**：安装包提供 `.desktop` 与 hicolor 图标，可从应用菜单或软件中心启动（自动打开终端）。
- **三平台工作流**：GitHub Actions + CNB 流水线，源码与 Release 同步到 GitHub、Gitee、CNB。
- **纯 Go TUI**：完整使用 Bubble Tea v2，不依赖 `fzf` 或任何外部选择器；唯一可选外部进程是图片兜底的 `chafa`。

## 功能

- 自动识别 Debian/Ubuntu、Arch、Fedora/RHEL 与 openSUSE，以及 `x86_64` / `aarch64` / `loongarch64` / `riscv64` 架构。
- 分类浏览、搜索、图标预览和长简介展开。
- Debian 使用 `aptss/aria2` 完成镜像下载、软件源摘要校验与断点续传，并持久化外围任务状态。
- 应用重启后自动识别完整包和 `.aria2` 中断任务；中断任务按 `D` 即可继续。
- 选中应用按 `P` 查询已安装版本与候选版本，只升级该应用；支持卸载和可选删除本地安装包。
- `sparkstore --version` 输出版本、提交与构建时间。

应用来源只保留 **Spark Store**。

## 依赖与下载流程

目录展示和 `/` 搜索使用 TUI 已读取的 Spark Store metadata，不会为每次按键启动外部进程；Debian 的实际软件包操作全部依赖 **`aptss | spark-store`**。`ca-certificates` 也是必需依赖；`sudo` 与用于图片兜底的 `chafa` 是推荐依赖。

Debian/Ubuntu/UOS/麒麟上的正常流程：

1. 根据系统架构和分类读取 Spark Store 公开元数据。
2. 从元数据中的 Deb 文件名取得真实包名，例如目录 `vscode` 对应 `code_…_amd64.deb`，实际包名是 `code`；同时核对架构，拒绝下载其他 CPU 的包。
3. 按 `D` 后在 `~/Downloads` 执行 `aptss download <包名>=<版本>`。`aptss/aria2` 负责镜像选择、仓库摘要校验和续传；TUI 会临时退出全屏界面以显示原生下载进度。
4. 外围任务状态保存在 `~/.config/sparkstore/tasks.json`。进程意外退出后，再次启动会结合最终文件和 `.aria2` 控制文件识别完成或中断状态。
5. 下载完成后按 `I` 或 `Enter`，执行 `aptss install <本地.deb> -y`；按 `Esc` 只保留安装包。
6. 选中应用按 `P` 会执行 `aptss policy <包名>`；发现更新时再次按 `P` 或 `Enter` 执行 `aptss install --only-upgrade <包名> -y`。

Arch、Fedora/RHEL 与 openSUSE 通过 Amber APM 安装；`P` 使用 `apm policy` 与 `apm install --only-upgrade`。详细命令与 metadata 字段见 [包管理链路调研](docs/package-backends.md)。

## 安装

### 一键安装（国内镜像默认）

```bash
bash <(curl -fsSL https://gitee.com/spark-store-project/spark-store-tui/raw/main/scripts/install-sparkstore.sh) --mirror gitee
```

安装器会检测发行版和 CPU 架构；Debian 系会先补齐并校验官方 `aptss`，再下载并校验对应发行包。新版优先使用 Release 的 `SHA256SUMS` 校验；缺少本机架构附件时，会询问是否从同一 tag 的源码构建。非交互执行默认使用 Gitee，源码回退构建默认使用 `https://goproxy.cn,direct`。

### Debian / Ubuntu / UOS / 麒麟

```bash
curl -LO https://gitee.com/spark-store-project/spark-store-tui/releases/download/v0.9.0/spark-store-tui_0.9.0-1_amd64.deb
sudo apt install ./spark-store-tui_0.9.0-1_amd64.deb
sudo aptss ssupdate
```

aarch64 把 `amd64` 换成 `arm64`，龙芯换成 `loong64`。通过软件中心或文件管理器双击 `.deb` 安装后，应用菜单会出现“星火终端助手”。

### RPM

```bash
sudo dnf install ./spark-store-tui-0.9.0-1.x86_64.rpm
# openSUSE:
sudo zypper install ./spark-store-tui-0.9.0-1.x86_64.rpm
```

### Arch / AUR

```bash
yay -S spark-store-tui
```

AUR 在本机架构构建，适合没有预编译附件的平台。

## 命令

```bash
sparkstore
sparkstore --version
sparkstore --bootstrap-images   # 显式安装 chafa（可选）
```

安装包另外提供 `SparkStore`、`SPARKSTORE` 和旧命令 `spark-store-tui`。

## 快捷键

进入应用页后：`/` 搜索、`[` / `]` 切换分类、`D` 下载/续传、`I` 安装、`P` 检查/更新所选应用、`U` 卸载、`E` 展开简介、`R` 刷新、`q` 退出。

## 发布物

| 渠道 | 产物 |
|---|---|
| Debian | `spark-store-tui_0.9.0-1_{amd64,arm64,loong64}.deb` |
| RPM | `spark-store-tui-0.9.0-1.{x86_64,aarch64,loongarch64}.rpm` |
| 源码 | `spark-store-tui-source-0.9.0-r1.tar.gz` |
| 校验 | `SHA256SUMS` |
| AUR | `spark-store-tui`（本机构建） |

Release 同时发布到 GitHub、Gitee（`spark-store-project`）与 CNB（`yxxbc`）。历史 [`legacy/apt/`](legacy/apt/README.md) 与 [`legacy/rpm/`](legacy/rpm/README.md) 只是冻结在 0.7.2 的归档，不代表当前版本。

## 文档

- [架构与分层](docs/architecture.md)
- [数据接口与内部边界](docs/data-interfaces.md)
- [包管理与下载链路](docs/package-backends.md)
- [安全说明](SECURITY.md)
- [发布流程](packaging/README.md)

## 开发与构建

需要 Go 1.25 或更新版本。

```bash
go test ./...
go vet ./...
make check
make go-build          # 本机二进制 -> build/sparkstore
./build/sparkstore --version
```

Linux 上构建完整发布物（需要 `dpkg-deb` 与 `rpmbuild`）：

```bash
make dist              # deb + rpm + source + AUR + SHA256SUMS -> dist/
```

`make build` 只构建本机架构 `.deb`；`make source` 只生成源码包。CI 在 Ubuntu runner 上执行同样的 `scripts/build-dist.sh`。

## 许可证

GPL-3.0-only，见 [COPYING](COPYING)。
