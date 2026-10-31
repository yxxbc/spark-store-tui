package platform

import "testing"

func TestFamilyAndArchitecture(t *testing.T) {
	if got := Family("ubuntu", "debian"); got != "deb" {
		t.Fatalf("family = %q, want deb", got)
	}
	if got := ArchitectureFromGOARCH("arm64"); got != "aarch64" {
		t.Fatalf("architecture = %q, want aarch64", got)
	}
	if got := ArchitectureFromGOARCH("loong64"); got != "loongarch64" {
		t.Fatalf("architecture = %q, want loongarch64", got)
	}
	if got := StorePath("store", "aarch64"); got != "arm64-store" {
		t.Fatalf("store path = %q", got)
	}
	if got := StorePath("store", "loongarch64"); got != "loong64-store" {
		t.Fatalf("store path = %q, want loong64-store", got)
	}
}

func TestValidateArchitecture(t *testing.T) {
	if err := ValidateArchitecture("amd64-store", "code_1.0_amd64.deb"); err != nil {
		t.Fatalf("matching architecture rejected: %v", err)
	}
	if err := ValidateArchitecture("loong64-store", "code_1.0_amd64.deb"); err == nil {
		t.Fatal("expected incompatible architecture error")
	}
	if err := ValidateArchitecture("amd64-store", "source.tar.gz"); err != nil {
		t.Fatalf("unmarked file rejected: %v", err)
	}
	if got := ArchitectureInFilename("spark-store-tui-0.9.0-1.aarch64.rpm"); got != "arm64" {
		t.Fatalf("filename architecture = %q", got)
	}
}

func TestImagePreviewCommand(t *testing.T) {
	command, err := ImagePreviewCommand("deb")
	if err != nil || command != "sudo apt-get update && sudo apt-get install -y chafa" {
		t.Fatalf("command = %q, err = %v", command, err)
	}
}
