package service

import (
	"strings"
	"testing"

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
}
