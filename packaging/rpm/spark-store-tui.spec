Name:           spark-store-tui
Version:        0.9.0
Release:        1%{?dist}
Summary:        Native terminal UI for Spark Store software management

License:        GPL-3.0-only
URL:            https://github.com/Xynrin/spark-store-tui
Source0:        https://github.com/Xynrin/%{name}/releases/download/v%{version}/%{name}-source-%{version}-r1.tar.gz

BuildRequires:  go >= 1.25
Requires:       ca-certificates
Recommends:     chafa
Recommends:     sudo

# Go binaries are already self-contained.  The host strip/objdump tools cannot
# inspect cross-compiled aarch64 binaries on the x86_64 release runner.
%global __brp_strip %{nil}
%global __brp_strip_comment_note %{nil}
%global debug_package %{nil}

%ifarch x86_64
%global sparkstore_goarch amd64
%endif
%ifarch aarch64
%global sparkstore_goarch arm64
%endif
%ifarch loongarch64
%global sparkstore_goarch loong64
%endif
%ifarch riscv64
%global sparkstore_goarch riscv64
%endif

%description
Spark Store TUI is a native Go terminal interface for browsing official Spark
Store metadata. On RPM systems, application installation, selected-application
updates and removal are delegated to Amber APM after explicit confirmation.

%prep
%autosetup -n %{name}-source-%{version}

%build
CGO_ENABLED=0 GOOS=linux GOARCH=%{sparkstore_goarch} go build -trimpath -buildvcs=false -o sparkstore ./cmd/sparkstore

%install
install -Dm0755 sparkstore %{buildroot}%{_libexecdir}/sparkstore/sparkstore
install -Dm0755 packaging/debian/package-root/usr/bin/sparkstore %{buildroot}%{_bindir}/sparkstore
ln -s sparkstore %{buildroot}%{_bindir}/SparkStore
ln -s sparkstore %{buildroot}%{_bindir}/SPARKSTORE
ln -s sparkstore %{buildroot}%{_bindir}/spark-store-tui
install -Dm0644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -Dm0644 COPYING %{buildroot}%{_licensedir}/%{name}/COPYING
install -Dm0644 packaging/debian/package-root/usr/share/applications/sparkstore.desktop %{buildroot}%{_datadir}/applications/sparkstore.desktop
install -Dm0644 packaging/debian/package-root/usr/share/icons/hicolor/256x256/apps/sparkstore.png %{buildroot}%{_datadir}/icons/hicolor/256x256/apps/sparkstore.png

%files
%license %{_licensedir}/%{name}/COPYING
%doc %{_docdir}/%{name}/README.md
%{_libexecdir}/sparkstore/sparkstore
%{_bindir}/sparkstore
%{_bindir}/SparkStore
%{_bindir}/SPARKSTORE
%{_bindir}/spark-store-tui
%{_datadir}/applications/sparkstore.desktop
%{_datadir}/icons/hicolor/256x256/apps/sparkstore.png

%post
update-desktop-database /usr/share/applications >/dev/null 2>&1 || :

%postun
update-desktop-database /usr/share/applications >/dev/null 2>&1 || :

%changelog
* Sat Oct 31 2026 Xynrin <xynrin@163.com> - 0.9.0-1
- Rewrite internal catalog, download, package backend and application layers
- Add native version reporting and unified three-platform release workflow

* Mon Jul 13 2026 Xynrin <xynrin@163.com> - 0.8.3-4
- Add selected-application update checks through Amber APM

* Mon Jul 13 2026 Xynrin <xynrin@163.com> - 0.8.3-3
- Resolve Amber package names from the published Debian asset filename

* Mon Jul 13 2026 Xynrin <xynrin@163.com> - 0.8.3-2
- Install Spark catalog applications through Amber APM on RPM systems

* Mon Jul 13 2026 Xynrin <xynrin@163.com> - 0.8.3-1
- Fix LoongArch metadata routing and package architecture validation
