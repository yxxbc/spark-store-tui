# spark-store-tui v0.9.0

0.9.0 是一次内部架构重写，用户可见的目录、下载与安装流程保持不变。

## 架构

- 新增 `internal/app` 用例层，统一目录加载、下载、安装、更新、卸载与状态查询。
- 新增 `internal/packages` 包后端抽象：Debian 使用 aptss，Arch/RPM/openSUSE 使用 Amber APM。
- 新增 `internal/config` 与 `internal/version`；`sparkstore --version` 输出版本、提交与构建时间。
- UI 只依赖 `domain` 与 `preview`，通过 `Service`/`Terminal` 端口与业务层交互。
- 合并 Metalink 与 aptss 下载任务的重复逻辑，中断恢复行为保持一致。

## 安全

- 所有包操作使用参数列表，不经过 shell；非 root 统一 `sudo -- <backend>`。
- 包名与包文件名白名单校验；删除安装包只允许下载目录内。
- Metalink 限制 1 MiB、拒绝非 http(s)/带凭据 URL、路径穿越文件名。
- 默认只接受 HTTPS；有 SHA-256 时优先校验，否则使用 SHA-1 做 HTTPS 传输完整性检查。
- 远程图片限制 12 MiB、要求 `image/*`、缓存目录 `0700`。
- Release 提供 `SHA256SUMS`，安装器优先按它校验。

## 体验

- 图片预览优先使用 Kitty/iTerm2/WezTerm/Ghostty 原生协议直出原图，其余终端回退 chafa。
- 安装包新增 `.desktop` 与 hicolor 图标，可从应用菜单或软件中心启动。
- 国内用户：一键安装默认 Gitee，源码回退默认 `goproxy.cn`；Release 同步 Gitee/CNB。

## 发布

- Debian：`spark-store-tui_0.9.0-1_{amd64,arm64,loong64}.deb`
- RPM：`spark-store-tui-0.9.0-1.{x86_64,aarch64,loongarch64}.rpm`
- 源码：`spark-store-tui-source-0.9.0-r1.tar.gz`
- AUR：`pkgrel=1`
- 校验：`SHA256SUMS`

## 文档

- `docs/architecture.md`
- `docs/data-interfaces.md`
- `docs/package-backends.md`
- `SECURITY.md`
