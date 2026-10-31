# 构建 spark-store-tui 0.9.1

完整打包与发布说明见 [`packaging/README.md`](packaging/README.md)。

在 Linux 或 WSL（Go 1.26+、`dpkg-deb`；构建 RPM 还需要 `rpmbuild`）中：

```bash
go test ./...
go vet ./...
make check
make build
sudo apt install ./dist/spark-store-tui_0.9.1-1_$(dpkg --print-architecture).deb
```

`make build` 通过 `scripts/build-dist.sh` 在 Linux 临时目录中生成原生架构 `.deb`，因此仓库位于 WSL 的 `/mnt/c` 时也不会受 NTFS 权限限制。

构建全部发布物：

```bash
make dist
```

产物位于 `dist/`，包含 Deb、RPM、源码包、AUR 模板和 `SHA256SUMS`。生成的 Deb 声明 `aptss | spark-store` 为应用管理后端依赖；直接 `apt install` 前需已安装官方 `aptss` 或完整的 Spark Store。面向普通用户推荐使用 `README.md` 中的一键安装命令，它会自动下载、校验 `aptss` 并初始化软件源。

仅构建本机二进制：

```bash
make go-build
./build/sparkstore --version
```
