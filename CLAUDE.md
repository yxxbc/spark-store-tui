# CLAUDE.md

Read `AGENTS.md` for repository roles, architecture rules, security constraints
and release policy.

Quick commands:

```bash
go test -race ./...
go vet ./...
make check
make go-build
./build/sparkstore --version
scripts/test-tui-macos.sh
```

Repository roles: Gitee is the mainline, GitHub builds/publishes packages and
serves as the overseas mirror, CNB is a mirror. Do not tag or publish unless the
maintainer explicitly asks.
