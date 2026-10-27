package where

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/v2rayA/v2rayA/common"
	"github.com/v2rayA/v2rayA/conf"
)

// Variant identifies which core binary answered.
type Variant string

const (
	// V2rayaCore is the merged v2raya-core binary (xray-core + MultiObservatory).
	// Binary name: v2raya_core
	V2rayaCore Variant = "V2rayaCore"
	// Xray is an xray-core binary named xray.
	Xray Variant = "Xray"
	// V2ray is a v2fly v2ray-core binary, 4.x or 5.x, named v2ray.
	V2ray Variant = "V2ray"
)

var NotFoundErr = fmt.Errorf("not found")

// ServiceNameList is the search order. v2raya_core stays first when it is
// installed beside a distribution v2ray or xray.
var ServiceNameList = []string{"v2raya_core", "xray", "v2ray"}

var v2rayVersion struct {
	version    string
	variant    Variant
	binPath    string
	lastUpdate time.Time
	mu         sync.Mutex
}

// GetV2rayServiceVersion returns the core binary's variant and version.
func GetV2rayServiceVersion() (variant Variant, ver string, err error) {
	// cache for 10 seconds
	v2rayVersion.mu.Lock()
	defer v2rayVersion.mu.Unlock()
	if time.Since(v2rayVersion.lastUpdate) < 10*time.Second && v2rayVersion.version != "" {
		return v2rayVersion.variant, v2rayVersion.version, nil
	}

	v2rayPath, err := GetV2rayBinPath()
	if err != nil || len(v2rayPath) == 0 {
		if err == nil {
			err = coreNotFound(nil)
		}
		return "", "", err
	}

	variant, ver, err = readCoreVersion(v2rayPath)
	if err != nil {
		return "", "", err
	}

	v2rayVersion.version = ver
	v2rayVersion.variant = variant
	v2rayVersion.binPath = v2rayPath
	v2rayVersion.lastUpdate = time.Now()
	return variant, ver, nil
}

func GetV2rayBinPath() (string, error) {
	v2rayBinPath := conf.GetEnvironmentConfig().V2rayBin
	if v2rayBinPath == "" {
		return getV2rayBinPathAnyway()
	}
	if _, err := os.Stat(v2rayBinPath); err != nil {
		return "", coreNotFound([]string{v2rayBinPath + " (--v2ray-bin)"})
	}
	return v2rayBinPath, nil
}

func getV2rayBinPathAnyway() (path string, err error) {
	var looked []string
	for _, target := range ServiceNameList {
		found, tried, findErr := getV2rayBinPath(target)
		looked = append(looked, tried...)
		if findErr == nil {
			return found, nil
		}
	}
	return "", coreNotFound(looked)
}

func coreNotFound(looked []string) error {
	where := "next to v2rayA and in PATH"
	if len(looked) > 0 {
		where = strings.Join(looked, ", ")
	}
	return common.Coded("CORE_NOT_FOUND", fmt.Errorf("%w: looked for v2raya_core, xray and v2ray in %s; set --v2ray-bin to the executable", NotFoundErr, where), map[string]interface{}{
		"paths": where,
	})
}

func getV2rayBinPath(target string) (path string, looked []string, err error) {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(target), ".exe") {
		target += ".exe"
	}
	// 先从可执行文件同目录下查找
	if exe, err := os.Executable(); err == nil {
		pa := filepath.Join(filepath.Dir(exe), target)
		looked = append(looked, pa)
		if _, err := os.Stat(pa); err == nil {
			return pa, looked, nil
		}
	}
	// 再从 PATH 环境变量里查找
	looked = append(looked, target+" (PATH)")
	if pa, err := exec.LookPath(target); err == nil {
		return pa, looked, nil
	}
	return "", looked, NotFoundErr
}

// parseCoreVersion reads the first line of `core version`.
// v2raya_core prints "V2RAYA_CORE <version>", xray prints "Xray <version>",
// and v2ray 4.x and 5.x print "V2Ray <version>".
func parseCoreVersion(output string) (variant Variant, ver string, err error) {
	line := strings.TrimSpace(output)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", fmt.Errorf("cannot parse version from output: %q", output)
	}
	switch strings.ToUpper(fields[0]) {
	case "XRAY":
		variant = Xray
	case "V2RAY":
		variant = V2ray
	case "V2RAYA_CORE":
		variant = V2rayaCore
	default:
		variant = V2rayaCore
	}
	return variant, fields[1], nil
}

func readCoreVersion(path string) (variant Variant, ver string, err error) {
	out, killed, runErr := execCore(path, "version")
	if variant, ver, err = parseCoreVersion(out); err == nil {
		return variant, ver, nil
	}
	// A binary that does not exit (the version argument was ignored) is not
	// asked again. v2ray 4.x exits from an unknown `version` subcommand and
	// answers `-version`.
	if !killed {
		out, _, runErr = execCore(path, "-version")
		if variant, ver, err = parseCoreVersion(out); err == nil {
			return variant, ver, nil
		}
	}
	if strings.TrimSpace(out) == "" && runErr != nil {
		return "", "", runErr
	}
	if err == nil {
		err = fmt.Errorf("cannot parse version from output: %q", out)
	}
	return "", "", err
}

func execCore(path, arg string) (output string, killed bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, arg)
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	cmd.Stderr = buf
	err = cmd.Run()
	killed = errors.Is(ctx.Err(), context.DeadlineExceeded)
	return buf.String(), killed, err
}
