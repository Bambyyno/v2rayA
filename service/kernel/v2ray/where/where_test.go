package where

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/v2rayA/v2rayA/conf"
)

func TestParseCoreVersion(t *testing.T) {
	cases := []struct {
		out     string
		variant Variant
		ver     string
	}{
		{"V2Ray 4.45.2 (V2Fly, a community-driven edition of V2Ray.) Custom\n", V2ray, "4.45.2"},
		{"V2Ray 5.16.1 (V2Fly, a community-driven edition of V2Ray.)\n", V2ray, "5.16.1"},
		{"Xray 1.8.24 (Xray, Penetrates Everything.) Custom\n", Xray, "1.8.24"},
		{"V2RAYA_CORE debug (based on xray-core 1.8.24)\n", V2rayaCore, "debug"},
	}
	for _, c := range cases {
		variant, ver, err := parseCoreVersion(c.out)
		if err != nil || variant != c.variant || ver != c.ver {
			t.Errorf("%q: got %s %s %v", c.out, variant, ver, err)
		}
	}
	if _, _, err := parseCoreVersion("nope"); err == nil {
		t.Fatal("unparseable output must fail")
	}
}

func resetVersionCache(t *testing.T) {
	t.Helper()
	v2rayVersion.mu.Lock()
	v2rayVersion.version = ""
	v2rayVersion.variant = ""
	v2rayVersion.binPath = ""
	v2rayVersion.err = nil
	v2rayVersion.lastUpdate = time.Time{}
	v2rayVersion.mu.Unlock()
}

func TestServiceVersionFromBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "xray")
	marker := filepath.Join(dir, "ran")
	// The script would claim to be v2raya_core. The file name is enough:
	// this package's xray is not run just to print a version.
	script := "#!/bin/sh\necho ran > '" + marker + "'\necho 'V2RAYA_CORE 9.9.9'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env := conf.GetEnvironmentConfig()
	prev := env.V2rayBin
	env.V2rayBin = bin
	t.Cleanup(func() { env.V2rayBin = prev })
	resetVersionCache(t)

	variant, ver, err := GetV2rayServiceVersion()
	if err != nil || variant != Xray || ver != "" {
		t.Fatalf("got %s %s %v", variant, ver, err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatal("xray was executed to read its version")
	}
}

func TestV2rayaCoreVersionStillComesFromTheBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "v2raya_core")
	script := "#!/bin/sh\necho 'V2RAYA_CORE 2.5.10 (based on xray-core 25.10.15)'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env := conf.GetEnvironmentConfig()
	prev := env.V2rayBin
	env.V2rayBin = bin
	t.Cleanup(func() { env.V2rayBin = prev })
	resetVersionCache(t)

	variant, ver, err := GetV2rayServiceVersion()
	if err != nil || variant != V2rayaCore || ver != "2.5.10" {
		t.Fatalf("got %s %s %v", variant, ver, err)
	}
}

func TestKilledVersionProbeIsNotAVersion(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "v2raya_core")
	script := "#!/bin/sh\nkill -9 $$\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env := conf.GetEnvironmentConfig()
	prev := env.V2rayBin
	env.V2rayBin = bin
	t.Cleanup(func() { env.V2rayBin = prev })
	resetVersionCache(t)

	variant, ver, err := GetV2rayServiceVersion()
	if !errors.Is(err, ErrProbeKilled) || variant != V2rayaCore || ver != "" {
		t.Fatalf("got %s %s %v", variant, ver, err)
	}
}

func TestMissingCoreNamesWhereItLooked(t *testing.T) {
	env := conf.GetEnvironmentConfig()
	prev := env.V2rayBin
	missing := filepath.Join(t.TempDir(), "missing-core")
	env.V2rayBin = missing
	t.Cleanup(func() { env.V2rayBin = prev })
	resetVersionCache(t)

	_, _, err := GetV2rayServiceVersion()
	if err == nil || !errors.Is(err, NotFoundErr) || !strings.Contains(err.Error(), missing) {
		t.Fatalf("error = %v", err)
	}
}

func TestSearchReportsNamesAndPath(t *testing.T) {
	env := conf.GetEnvironmentConfig()
	prev := env.V2rayBin
	env.V2rayBin = ""
	t.Cleanup(func() { env.V2rayBin = prev })

	_, err := getV2rayBinPathAnyway()
	if err == nil {
		t.Skip("a core binary is installed on this machine")
	}
	if !errors.Is(err, NotFoundErr) || !strings.Contains(err.Error(), "v2ray") || !strings.Contains(err.Error(), "PATH") {
		t.Fatalf("error = %v", err)
	}
}
