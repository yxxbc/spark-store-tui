package download

import "testing"

func TestSafeFilename(t *testing.T) {
	valid := []string{"code_1.0_amd64.deb", "app.tar.gz", "包_1.0.deb"}
	for _, name := range valid {
		if !safeFilename(name) {
			t.Fatalf("valid filename rejected: %q", name)
		}
	}
	invalid := []string{"", ".", "..", "../evil.deb", "dir/evil.deb", "dir\\evil.deb", "bad\x1bname.deb", "bad\x00name"}
	for _, name := range invalid {
		if safeFilename(name) {
			t.Fatalf("unsafe filename accepted: %q", name)
		}
	}
}
