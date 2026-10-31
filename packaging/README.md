# 0.9.0 打包与发布

发布版本统一来自根目录 `VERSION`（当前 `0.9.0`），修订号来自 `packaging/revisions.env`。

| 渠道 | 产物 |
|---|---|
| GitHub / Gitee / CNB Release | `spark-store-tui-source-0.9.0-r1.tar.gz`、各架构 `.deb`、`.rpm`、`SHA256SUMS` |
| Debian | `spark-store-tui_0.9.0-1_{amd64,arm64,loong64}.deb` |
| RPM | `spark-store-tui-0.9.0-1.{x86_64,aarch64,loongarch64}.rpm` |
| AUR | `dist/aur/PKGBUILD`、`dist/aur/.SRCINFO` |

## 一条命令构建全部产物

在 Linux（或装有 `dpkg-deb` / `rpmbuild` 的构建容器）中：

```bash
make check
make dist        # 等价于 scripts/build-dist.sh
```

产物写入 `dist/`：

```text
spark-store-tui_0.9.0-1_amd64.deb
spark-store-tui_0.9.0-1_arm64.deb
spark-store-tui_0.9.0-1_loong64.deb
spark-store-tui-0.9.0-1.x86_64.rpm
spark-store-tui-0.9.0-1.aarch64.rpm
spark-store-tui-0.9.0-1.loongarch64.rpm
spark-store-tui-source-0.9.0-r1.tar.gz
SHA256SUMS
aur/PKGBUILD
aur/.SRCINFO
```

可选参数：

```bash
scripts/build-dist.sh --skip-rpm
scripts/build-dist.sh --skip-deb --skip-source
scripts/build-dist.sh --deb-archs "amd64 arm64"
scripts/build-dist.sh --version 0.9.0
```

`make build` 只构建本机架构 Deb，`make source` 只生成源码包。

## Debian

`packaging/debian/package-root` 是包根，包含：

- `DEBIAN/control`（构建脚本按版本/架构渲染）
- `usr/bin/sparkstore` 启动脚本
- `usr/lib/sparkstore/sparkstore` 原生二进制（构建时写入）
- `usr/share/applications/sparkstore.desktop`
- `usr/share/icons/hicolor/256x256/apps/sparkstore.png`

安装后可从桌面应用菜单启动“星火终端助手”（`Terminal=true`）。

## RPM

`scripts/build-dist.sh` 会用临时 spec 渲染版本、Release 与 Source0，然后：

```bash
rpmbuild -bb --nodeps --target <x86_64|aarch64|loongarch64> ...
```

spec 同时安装 `.desktop` 和图标，并在 `%post`/`%postun` 刷新桌面数据库。

## AUR

`scripts/build-dist.sh` 从同一源码包计算 SHA-256，渲染 `dist/aur/PKGBUILD` 与 `.SRCINFO`。发布：

```bash
scripts/publish-aur.sh
```

需要 AUR SSH 推送权限，并先构建 `dist/aur`。

## 发布到三平台

```bash
# 本地预演
scripts/sync-all.sh --dry-run

# 实际发布（Gitee + CNB；--github 同时发 GitHub）
scripts/sync-all.sh --build
```

GitHub Actions 的 `release.yml` 在 `v*` tag 上构建全部产物并发布 GitHub，随后调用 `repo-all-sync` 发布 Gitee/CNB；`mirror.yml` 负责 main/tag 源码镜像。CNB 的 `.cnb.yml` 也包含 tag 发布流水线。

## 版本一致性

```bash
make release-check   # 校验 VERSION、spec、control、AUR 模板与旧路径残留
```

修改上游版本时只改 `VERSION`；只改打包时改 `packaging/revisions.env`。
