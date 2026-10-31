package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	tea "charm.land/bubbletea/v2"
	"github.com/yxxbc/spark-store-tui/internal/app"
	"github.com/yxxbc/spark-store-tui/internal/catalog"
	"github.com/yxxbc/spark-store-tui/internal/config"
	"github.com/yxxbc/spark-store-tui/internal/download"
	"github.com/yxxbc/spark-store-tui/internal/packages"
	"github.com/yxxbc/spark-store-tui/internal/platform"
	"github.com/yxxbc/spark-store-tui/internal/preview"
	"github.com/yxxbc/spark-store-tui/internal/state"
	"github.com/yxxbc/spark-store-tui/internal/ui"
	"github.com/yxxbc/spark-store-tui/internal/version"
)

func main() {
	bootstrapImages := flag.Bool("bootstrap-images", false, "install the optional terminal image preview dependency for this distribution")
	showVersion := flag.Bool("version", false, "print version information and exit")
	previewImage := flag.String("preview-image", "", "render an image URL outside the TUI and exit")
	previewWait := flag.Bool("preview-wait", false, "wait for Enter after rendering the image")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return
	}
	if *previewImage != "" {
		if err := preview.DisplayURL(*previewImage, 40, 20, *previewWait); err != nil {
			fmt.Fprintln(os.Stderr, "sparkstore:", err)
			os.Exit(1)
		}
		return
	}

	host := platform.Detect()
	if *bootstrapImages {
		if platform.HasImagePreview() {
			fmt.Println("terminal image preview dependency is already available")
			return
		}
		if err := platform.RunImagePreviewBootstrap(host); err != nil {
			fmt.Fprintln(os.Stderr, "sparkstore:", err)
			os.Exit(1)
		}
		return
	}

	paths := config.Load()
	tasks := state.NewTaskStore(paths.TasksFile())

	sources := catalog.BuiltinSources()
	if len(sources) == 0 {
		fmt.Fprintln(os.Stderr, "sparkstore: no catalog sources configured")
		os.Exit(1)
	}
	catalogs := map[string]catalog.Provider{
		sources[0].ID: catalog.SparkMetadataProvider{
			Catalog:         sources[0],
			DefaultArchPath: host.SparkArchPath,
		},
	}

	var downloader app.Downloader
	if host.Family == "deb" {
		downloader = download.NewAPTSSDownloader(tasks, paths.DownloadDir)
	} else {
		downloader = download.NewMetalinkDownloader(nil, tasks, paths.DownloadDir)
	}

	backend, backendErr := packages.For(host)
	if backendErr != nil && runtime.GOOS == "linux" {
		// The TUI must still browse metadata on unsupported hosts; package
		// operations report the same error when the user asks for them.
		fmt.Fprintln(os.Stderr, "sparkstore: package operations unavailable:", backendErr)
	}

	service := app.New(app.Deps{
		Sources:    sources,
		Catalogs:   catalogs,
		Backend:    backend,
		Downloader: downloader,
		Host:       host,
		Paths:      paths,
	})

	recovered, recoveryErr := service.Recover()
	if recoveryErr != nil {
		fmt.Fprintln(os.Stderr, "sparkstore: could not recover download state:", recoveryErr)
	}

	program := tea.NewProgram(ui.New(service, terminal{}).WithRecoveredDownloads(recovered))
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "sparkstore:", err)
		os.Exit(1)
	}
}
