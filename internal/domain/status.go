package domain

// PackageStatus is the package manager's view of an application. The catalog
// version is deliberately not used: the backend's candidate version is what an
// install or upgrade would actually select.
type PackageStatus struct {
	PackageName      string
	InstalledVersion string
	CandidateVersion string
	Installed        bool
	UpdateAvailable  bool
}
