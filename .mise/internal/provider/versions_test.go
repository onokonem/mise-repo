package provider

import "testing"

func TestLoadVersions(t *testing.T) {
	v, err := LoadVersions("../../../versions.toml")
	if err != nil {
		t.Fatal(err)
	}
	if len(v.GoVersions) != 3 {
		t.Fatalf("go_versions: got %d, want 3", len(v.GoVersions))
	}
	if v.GoTags["1.26"] == "" {
		t.Fatal("no [tools.go] tag for 1.26")
	}
	gci, ok := v.Built["golangci-lint"]
	if !ok {
		t.Fatal("no [tools.golangci-lint]")
	}
	if len(gci) < 2 {
		t.Errorf("golangci-lint versions: got %d, want >= 2 (v1 + v2)", len(gci))
	}
	if len(v.Built) < 27 {
		t.Errorf("built tools: got %d, want >= 27", len(v.Built))
	}
	// every entry of every tool has all fields
	for name, entries := range v.Built {
		for _, b := range entries {
			if b.Version == "" || b.Path == "" || b.Binary == "" {
				t.Errorf("tools.%s entry incomplete: %+v", name, b)
			}
		}
	}
	if got := CleanVer("v2.12.2"); got != "2.12" {
		t.Errorf("CleanVer(v2.12.2) = %q, want 2.12", got)
	}
	if got := CleanVer("2025.1.1"); got != "2025.1" {
		t.Errorf("CleanVer(2025.1.1) = %q, want 2025.1", got)
	}
}
