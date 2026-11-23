package v2ray

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/kernel/serverObj"
	"github.com/v2rayA/v2rayA/kernel/v2ray/where"
)

func useLeastPing(t *testing.T) {
	t.Helper()
	previous := configure.GetOutboundSetting("proxy")
	t.Cleanup(func() { _ = configure.SetOutboundSetting("proxy", previous) })
	setting := configure.DefaultOutboundSetting()
	setting.Type = configure.LeastPing
	if err := configure.SetOutboundSetting("proxy", setting); err != nil {
		t.Fatal(err)
	}
}

func socksInfo(outbound, server string) serverInfo {
	return serverInfo{
		Info:         &serverObj.SOCKS{Name: server, Server: server, Port: 1080, Protocol: "socks5"},
		OutboundName: outbound,
	}
}

func TestBalancedSnapshotNodeIPs(t *testing.T) {
	tmpl := &Template{}
	data := NewServerData([]serverInfo{socksInfo("proxy", "192.0.2.1"), socksInfo("proxy", "192.0.2.2")})
	if _, _, err := tmpl.resolveOutbounds(data); err != nil {
		t.Fatal(err)
	}
	ips := collectNodeIPs(tmpl)
	sort.Strings(ips)
	if want := []string{"192.0.2.1", "192.0.2.2"}; !reflect.DeepEqual(ips, want) {
		t.Fatalf("snapshot bypass IPs: %v, want %v", ips, want)
	}
}

func TestLeastPingOnXrayUsesNativeObservatory(t *testing.T) {
	useLeastPing(t)
	tmpl := &Template{Variant: where.Xray}
	data := NewServerData([]serverInfo{socksInfo("proxy", "a"), socksInfo("proxy", "b")})
	if _, _, err := tmpl.resolveOutbounds(data); err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.SetAPI(data); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tmpl.Close() })
	if tmpl.MultiObservatory != nil {
		t.Fatal("stock xray was given multiObservatory, which it does not load")
	}
	if tmpl.Observatory == nil || len(tmpl.Observatory.SubjectSelector) != 2 ||
		tmpl.Observatory.ProbeURL == "" || tmpl.Observatory.ProbeInterval == "" {
		t.Fatalf("observatory: %+v", tmpl.Observatory)
	}
	if len(tmpl.Routing.Balancers) != 1 || tmpl.Routing.Balancers[0].Strategy.Type != "leastping" ||
		tmpl.Routing.Balancers[0].FallbackTag == "" {
		t.Fatalf("balancer: %+v", tmpl.Routing.Balancers)
	}
	raw, err := json.Marshal(tmpl.Observatory)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(`"settings"`)) || !bytes.Contains(raw, []byte(`"probeURL"`)) {
		t.Fatalf("observatory is not the flat xray object: %s", raw)
	}
}

func TestLeastPingOnV2rayaCoreKeepsMultiObservatory(t *testing.T) {
	useLeastPing(t)
	tmpl := &Template{Variant: where.V2rayaCore}
	data := NewServerData([]serverInfo{socksInfo("proxy", "a"), socksInfo("proxy", "b")})
	if _, _, err := tmpl.resolveOutbounds(data); err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.SetAPI(data); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tmpl.Close() })
	if tmpl.Observatory != nil || tmpl.MultiObservatory == nil || len(tmpl.MultiObservatory.Observers) != 1 {
		t.Fatalf("observatory=%+v multi=%+v", tmpl.Observatory, tmpl.MultiObservatory)
	}
	if len(tmpl.Routing.Balancers) != 1 || tmpl.Routing.Balancers[0].FallbackTag == "" {
		t.Fatalf("balancer: %+v", tmpl.Routing.Balancers)
	}
}
