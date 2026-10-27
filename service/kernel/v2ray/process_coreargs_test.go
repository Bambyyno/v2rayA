package v2ray

import (
	"strings"
	"testing"

	"github.com/v2rayA/v2rayA/kernel/v2ray/where"
)

func TestCoreCommandArgs(t *testing.T) {
	v4 := coreCommandArgs("/usr/bin/v2ray", "/tmp/config.json", "/tmp/conf", where.V2ray, "4.45.2")
	if strings.Join(v4, " ") != "/usr/bin/v2ray -config=/tmp/config.json" {
		t.Fatalf("v2ray 4: %v", v4)
	}
	v5 := coreCommandArgs("/usr/bin/v2ray", "/tmp/config.json", "/tmp/conf", where.V2ray, "5.16.1")
	if strings.Join(v5, " ") != "/usr/bin/v2ray run --config=/tmp/config.json --confdir=/tmp/conf" {
		t.Fatalf("v2ray 5: %v", v5)
	}
	xray := coreCommandArgs("/usr/bin/xray", "/tmp/config.json", "", where.Xray, "1.8.24")
	if strings.Join(xray, " ") != "/usr/bin/xray run --config=/tmp/config.json" {
		t.Fatalf("xray: %v", xray)
	}
	if got := coreAssetEnv(where.V2ray, "/usr/share/v2ray"); got != "V2RAY_LOCATION_ASSET=/usr/share/v2ray" {
		t.Fatalf("v2ray asset env: %s", got)
	}
	if got := coreAssetEnv(where.Xray, "/usr/share/xray"); got != "XRAY_LOCATION_ASSET=/usr/share/xray" {
		t.Fatalf("xray asset env: %s", got)
	}
}
