package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v2rayA/v2rayA/conf"
	"github.com/v2rayA/v2rayA/kernel/v2ray/where"
)

func TestAcceptCoreVersion(t *testing.T) {
	if err := acceptCoreVersion(where.Xray, "1.8.24", "2.2.7"); err != nil {
		t.Fatalf("xray: %v", err)
	}
	if err := acceptCoreVersion(where.V2ray, "4.45.2", "2.2.7"); err != nil {
		t.Fatalf("v2ray 4: %v", err)
	}
	if err := acceptCoreVersion(where.V2ray, "5.16.1", "2.2.7"); err != nil {
		t.Fatalf("v2ray 5: %v", err)
	}
	if err := acceptCoreVersion(where.V2rayaCore, "1.0.0", "2.0.0"); err == nil || !strings.Contains(err.Error(), "1.0.0") {
		t.Fatalf("v2raya_core mismatch: %v", err)
	}
	if err := acceptCoreVersion(where.V2rayaCore, "debug", "2.0.0"); err != nil {
		t.Fatalf("debug core: %v", err)
	}
	if err := acceptCoreVersion(where.V2rayaCore, "2.0.0", "2.0.0"); err != nil {
		t.Fatalf("matching core: %v", err)
	}
	if err := acceptCoreVersion(where.Xray, "", "2.5.10.20261027"); err != nil {
		t.Fatalf("xray without a printed version: %v", err)
	}
}

func TestNamedXrayIsAcceptedWithoutRunning(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "xray")
	marker := filepath.Join(dir, "ran")
	script := "#!/bin/sh\necho ran > '" + marker + "'\necho 'V2RAYA_CORE 0.0.1'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env := conf.GetEnvironmentConfig()
	prev := env.V2rayBin
	env.V2rayBin = bin
	t.Cleanup(func() { env.V2rayBin = prev })
	if err := CheckCoreVersionMatch(); err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatal("xray was executed to compare versions")
	}
}

func TestKilledProbeIsNotAVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "v2raya_core")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nkill -9 $$\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := conf.GetEnvironmentConfig()
	prev := env.V2rayBin
	env.V2rayBin = bin
	t.Cleanup(func() { env.V2rayBin = prev })
	if err := CheckCoreVersionMatch(); err != nil {
		t.Fatalf("killed probe reported as %v", err)
	}
}
