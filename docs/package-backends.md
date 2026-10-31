# 包管理与下载链路调研（0.9.1）

本文记录 `sparkstore` 实际使用的星火商店包管理路径。结论以 2026-10-31 的线上 metadata 与仓库代码为准。

## 结论速览

| 发行版 | 目录展示 | 下载 | 安装 / 更新 / 卸载 | 版本查询 |
|---|---|---|---|---|
| Debian / Ubuntu / UOS / 麒麟 | Spark metadata | `aptss download` | `aptss install` / `aptss remove` | `aptss policy` |
| Arch / Fedora / RHEL / openSUSE | Spark metadata | 不单独下载 | Amber APM `apm install` / `apm remove` | `apm policy` |

TUI 本身不直接调用 `apt`、`dnf`、`pacman`、`zypper` 安装星火应用，也不调用 `fzf`、`rg`、`find` 等外部进程完成交互。默认只读取公开 metadata；只有用户明确确认后才启动 `aptss` 或 `apm`。

## Debian：aptss

`aptss` 是星火商店维护的 Debian 包管理后端，负责星火软件源、镜像选择、aria2 续传和仓库摘要校验。

```bash
aptss ssupdate                              # 初始化/刷新星火软件源索引
aptss download <包名>=<版本>                # 镜像下载，aria2 续传
aptss install <本地.deb> -y                 # 安装已下载的包
aptss install --only-upgrade <包名> -y      # 只升级所选应用
aptss remove <包名> -y                      # 卸载
aptss policy <包名>                         # 查看 Installed / Candidate
```

- 包名以 metadata 的 `Filename` 前缀为准。例如目录 `vscode` 发布 `code_<version>_amd64.deb`，实际包名是 `code`。
- `aptss download` 使用星火软件源配置，不在 TUI 内自行选择镜像或拼接 URL。
- Deb 包依赖：`ca-certificates`、`aptss | spark-store`；推荐 `chafa`、`sudo`。

## Arch / Fedora / RHEL / openSUSE：Amber APM

```bash
apm install <包名> -y
apm install --only-upgrade <包名> -y
apm remove <包名> -y
apm policy <包名>
apm update
```

- 这些发行版不直接安装 Spark 的 `.deb`；由 Amber APM 提供兼容安装。
- 包名同样优先使用 Deb 文件名前缀，避免目录名与真实包名不一致。
- 缺少 `apm` 时，安装/卸载会明确报错；目录浏览与下载不受影响。

## 线上 metadata 接口

| 用途 | 地址 | 说明 |
|---|---|---|
| 分类 | `https://d.spark-app.store/store/categories.json` | `{id: {"zh": "...", "en": "..."}}` |
| 应用列表 | `https://d.spark-app.store/{arch}-store/{category}/applist.json` | JSON 数组 |
| Metalink | `https://d.spark-app.store/{arch}-store/{category}/{Pkgname}/{Filename}.metalink` | Metalink v3 |

`arch` 目录：`amd64-store`、`arm64-store`、`loong64-store`、`riscv64-store`。

`applist.json` 的关键字段：

```json
{
  "Name": "Visual Studio Code(vscode)",
  "Version": "1.141.0-1791276758",
  "Filename": "code_1.141.0-1791276758_amd64.deb",
  "Pkgname": "code",
  "Size": "238M",
  "More": "简介文本",
  "icons": "https://cdn.d.store.deepinos.org.cn/.../icon.png",
  "img_urls": "[\"https://cdn.d.store.deepinos.org.cn/.../screen_1.png\"]"
}
```

`icons` 与 `img_urls` 在不同条目中可能是 JSON 数组，也可能是“包含数组的 JSON 字符串”；客户端两种都接受。

- `icons` 是**应用图标**，用于应用标识和详情页兜底。
- `img_urls` 是**应用内界面截图**。详情页按标准商店排版同时显示应用图标和首张 `img_urls` 截图，不做切换；没有截图时只显示图标。

Metalink 当前发布 MD5、SHA-1 和空的 PGP 签名占位，**没有 SHA-256**。客户端的处理：

1. 有 SHA-256 时优先校验 SHA-256。
2. 否则校验 SHA-1，作为 HTTPS 传输完整性检查；SHA-1 不具备抗碰撞性，不作为长期真实性证明。
3. 拒绝非 `http(s)` URL、带用户信息的 URL、路径穿越文件名。
4. 默认只接受 HTTPS；确需兼容明文镜像时显式设置 `SPARKSTORE_ALLOW_HTTP=1`。
5. Metalink 响应限制为 1 MiB，避免解析被放大的 XML。

## 元数据新鲜度与同步

- 目录在启动、刷新、切换分类时实时请求，不使用离线 applist 缓存；请求带 `Cache-Control: no-cache`。
- 下载/更新前，如果目录已加载超过 10 分钟，先重新拉取目录再执行操作，避免用过期版本下载。
- applist 按包名去重；带路径分隔符的文件名直接丢弃。
- 包名以 Deb 文件名前缀为准，后端再调用 `policy` 获取真实的已安装/候选版本。
- 图片按 URL 哈希缓存 24 小时；过期会重新校验，失败时保留旧图而不是空白。

## 国内用户可用性

- metadata 与镜像均由星火官方域名提供，国内网络可直连；Metalink 中 `location="cn"`、`preference` 较高的镜像优先。
- Release 产物同时发布到 Gitee（`spark-store-project/spark-store-tui`）与 CNB（`yxxbc/spark-store-tui`）；安装脚本支持 `--mirror gitee`。
- 源码回退构建时，安装脚本默认使用 `GOPROXY=https://goproxy.cn,direct`（若用户未设置 `GOPROXY`）。
- 图形桌面通过 `.desktop` 入口启动：安装包会在应用菜单中生成“星火终端助手”，`Terminal=true` 会自动打开终端运行 `sparkstore`。

## 为什么不用 fzf

TUI 由 Bubble Tea v2 在进程内完整实现，列表、搜索、窗口滚动、确认状态机都是 Go 代码；没有 `fzf` 子进程，也没有 shell 拼接。唯一可选外部进程是 `chafa`，用于不支持原生图片协议时的终端图片兜底。
