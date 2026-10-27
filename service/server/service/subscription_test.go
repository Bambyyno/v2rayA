package service

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestSubscriptionBodyOverLimitIsRejected(t *testing.T) {
	client := &http.Client{Transport: updateTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", int(maxSubscriptionDocumentSize+1)))),
			Header:     make(http.Header),
		}, nil
	})}
	if _, _, err := ResolveSubscriptionWithClient("https://example.test/subscription", client); err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Fatalf("error = %v, want size limit", err)
	}
}

func TestSubscriptionCoreApplyErrorWrapsCause(t *testing.T) {
	cause := errors.New("reload failed")
	err := subscriptionCoreApplyError(cause)
	if !errors.Is(err, cause) || !strings.HasPrefix(err.Error(), "subscription stored, but the core could not apply it:") {
		t.Fatalf("error = %v", err)
	}
}

func TestSubscriptionUserInfoString(t *testing.T) {
	const gib = 1024 * 1024 * 1024
	cases := []struct {
		name string
		sui  SubscriptionUserInfo
		want string
	}{
		{"usage and expiry", SubscriptionUserInfo{Upload: 0, Download: 1041529159, Total: 100 * gib, Expire: time.Date(2026, 10, 4, 18, 59, 0, 0, time.UTC)},
			"Used 0.97 GiB / 100.00 GiB · Expires " + time.Date(2026, 10, 4, 18, 59, 0, 0, time.UTC).Local().Format("2006-01-02")},
		{"no total", SubscriptionUserInfo{Upload: gib, Download: gib, Total: -1}, "Used 2.00 GiB"},
		{"total only", SubscriptionUserInfo{Upload: -1, Download: -1, Total: 10 * gib}, "Total 10.00 GiB"},
		{"nothing", SubscriptionUserInfo{Upload: -1, Download: -1, Total: -1}, ""},
	}
	for _, c := range cases {
		if got := c.sui.String(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if c.sui.Known() != (c.want != "") {
			t.Errorf("%s: Known() disagrees with the rendered text", c.name)
		}
	}
}

func TestSubscriptionSendsStableDeviceID(t *testing.T) {
	var got http.Header
	body := base64.StdEncoding.EncodeToString([]byte("vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?type=tcp#real\n"))
	client := &http.Client{Transport: updateTransport(func(r *http.Request) (*http.Response, error) {
		got = r.Header.Clone()
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}
	infos, _, err := ResolveSubscriptionWithClient("https://example.test/sub", client)
	if err != nil || len(infos) != 1 || infos[0].GetHostname() != "1.2.3.4" {
		t.Fatalf("infos=%v err=%v", infos, err)
	}
	id := got.Get("x-hwid")
	if id == "" || id != got.Get("X-Device-Id") {
		t.Fatalf("device headers: %v", got)
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9=-]{10,64}$`).MatchString(id) {
		t.Fatalf("device id %q", id)
	}
	if id != subscriptionDeviceID() {
		t.Fatal("device id changed between calls")
	}
	ua := got.Get("User-Agent")
	if !strings.Contains(ua, "v2rayA/") || strings.Contains(strings.ToLower(ua), "happ") || strings.Contains(ua, "INCY") {
		t.Fatalf("user agent %q", ua)
	}
}

func TestSubscriptionPlaceholderIsNotImported(t *testing.T) {
	raw := "vless://b831381d-6324-4d53-ad4f-8cda48b30811@0.0.0.0:1?type=tcp#" + url.PathEscape("Приложение не поддерживается") +
		"\nvless://b831381d-6324-4d53-ad4f-8cda48b30811@0.0.0.0:1?type=tcp#" + url.PathEscape("Используйте HAPP или INCY") + "\n"
	body := base64.StdEncoding.EncodeToString([]byte(raw))
	client := &http.Client{Transport: updateTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	infos, _, err := ResolveSubscriptionWithClient("https://sub.example/S", client)
	if err == nil || len(infos) != 0 || !strings.Contains(err.Error(), "0.0.0.0") {
		t.Fatalf("infos=%v err=%v", infos, err)
	}
	if !strings.Contains(err.Error(), "HAPP") {
		t.Fatalf("error = %v", err)
	}
}

func TestSubscriptionDeviceLimitIsNotImported(t *testing.T) {
	raw := "vless://b831381d-6324-4d53-ad4f-8cda48b30811@0.0.0.0:1?type=tcp#" + url.PathEscape("Вы достигли лимита устройств") + "\n"
	body := base64.StdEncoding.EncodeToString([]byte(raw))
	header := make(http.Header)
	header.Set("X-Hwid-Active", "true")
	header.Set("X-Hwid-Max-Devices-Reached", "true")
	header.Set("X-Hwid-Limit", "true")
	client := &http.Client{Transport: updateTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     header,
		}, nil
	})}
	infos, _, err := ResolveSubscriptionWithClient("https://sub.example/S", client)
	if err == nil || len(infos) != 0 || !strings.Contains(err.Error(), "device limit") {
		t.Fatalf("infos=%v err=%v", infos, err)
	}
	if strings.Contains(err.Error(), "HAPP") {
		t.Fatalf("device limit was reported as a HAPP placeholder: %v", err)
	}
}

func TestSubscriptionDropsPlaceholderBesideRealServer(t *testing.T) {
	raw := "vless://b831381d-6324-4d53-ad4f-8cda48b30811@0.0.0.0:1?type=tcp#App%20not%20supported\n" +
		"vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?type=tcp#real\n"
	body := base64.StdEncoding.EncodeToString([]byte(raw))
	client := &http.Client{Transport: updateTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	infos, _, err := ResolveSubscriptionWithClient("https://sub.example/S", client)
	if err != nil || len(infos) != 1 || infos[0].GetHostname() != "1.2.3.4" {
		t.Fatalf("infos=%v err=%v", infos, err)
	}
}

func TestParseSubscriptionUserInfoHeader(t *testing.T) {
	sui := parseSubscriptionUserInfo("upload=0; download=1041529159; total=107374182400; expire=1791140340")
	if sui.Upload != 0 || sui.Download != 1041529159 || sui.Total != 107374182400 {
		t.Fatalf("fields: %+v", sui)
	}
	if sui.Expire.UTC().Format("2006-01-02 15:04") != "2026-10-04 18:59" {
		t.Fatalf("expire: %v", sui.Expire)
	}
	if !sui.Known() {
		t.Fatal("a header with fields must count as known")
	}
	empty := parseSubscriptionUserInfo("")
	if empty.Known() {
		t.Fatal("an empty header must not count as known")
	}
}
