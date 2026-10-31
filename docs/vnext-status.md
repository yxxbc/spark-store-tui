# 0.9.0 状态与已知限制

## 已完成

- 深度重写内部控制层：`app`、`catalog`、`download`、`packages`、`platform`、`config`、`version`。
- UI 只依赖 `domain` 与 `preview`，通过 `ui.Service` / `ui.Terminal` 端口与业务层交互。
- 下载任务生命周期去重：Metalink 与 aptss 共用 `taskLifecycle`。
- 包后端抽象：aptss 与 Amber APM 统一为 `packages.Backend`。
- 安全加固：`sudo --`、参数列表、文件名/包名白名单、下载目录边界、Metalink 1 MiB 限制、URL scheme 校验、HTTPS 默认、SHA-256/SHA-1 校验。
- 原图直出（Kitty/iTerm2/WezTerm/Ghostty）+ chafa 兜底。
- GUI 桌面入口：`.desktop` 与 hicolor 图标进入 Deb/RPM/AUR。
- 版本注入与 `sparkstore --version`。
- 单版本源 `VERSION` 与打包修订 `packaging/revisions.env`。
- `scripts/build-dist.sh` 生成 Deb/RPM/源码/AUR/SHA256SUMS。
- GitHub Actions CI、Release、Mirror 工作流；CNB 流水线；本地 `scripts/sync-all.sh`。

## 尚未完成

- TUI 内还没有下载队列、暂停、取消和历史页；Debian 下载时显示 aptss/aria2 原生进度。
- AppImage、Flatpak、Spark `.spk` 与 APM 审计安装还没有统一安装实现。
- 应用详情目前显示图标；截图图库、切换和懒加载尚未完成。
- 原生图片协议只覆盖 Kitty 与 iTerm2 系；Sixel 终端仍走 chafa。

## 已知问题

- Spark Metalink 目前只发布 MD5/SHA-1，没有 SHA-256；客户端用 SHA-1 做 HTTPS 传输完整性校验，不等同于发布者签名。
- 首次缓存远程图片仍受镜像网络影响，超过 8 秒会放弃显示；后续命中缓存会很快。
- 很窄的终端会启用单栏摘要模式，图片和完整详情会隐藏以保证可操作性。
- 已安装状态依赖 aptss/APM 的包数据库；AppImage 和 Flatpak 还没有统一识别。
- Gitee/CNB 发布需要对应 token；CNB 镜像为快照，不含完整 Git 历史。
