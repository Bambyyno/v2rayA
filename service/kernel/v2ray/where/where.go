package where

import (
	"bytes"
	"context"
	"debug/buildinfo"
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

// ErrProbeKilled means the version command was SIGKILL'd or timed out.
// That is not a version string, so it must not be reported as a mismatch.
var ErrProbeKilled = errors.New("core version probe was killed")

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
	err        error
	lastUpdate time.Time
	mu         sync.Mutex
}

// GetV2rayServiceVersion returns the core binary's variant and version.
func GetV2rayServiceVersion() (variant Variant, ver string, err error) {
	v2rayVersion.mu.Lock()
	defer v2rayVersion.mu.Unlock()

	v2rayPath, err := GetV2rayBinPath()
	if err != nil || len(v2rayPath) == 0 {
		if err == nil {
			err = coreNotFound(nil)
		}
		return "", "", err
	}
	// cache for 10 seconds, including an xray hit that has no version string
	// and a probe that was killed. A killed probe must not be run again.
	if v2rayVersion.binPath == v2rayPath && time.Since(v2rayVersion.lastUpdate) < 10*time.Second &&
		(v2rayVersion.version != "" || v2rayVersion.variant == Xray || v2rayVersion.err != nil) {
		return v2rayVersion.variant, v2rayVersion.version, v2rayVersion.err
	}

	// The Go build info and the file name already say which core this is.
	// xray's version is not compared with v2rayA, and spawning it only to
	// print that string is enough to be SIGKILL'd on a small router.
	fromInfo := variantFromBuildInfo(v2rayPath)
	fromName := variantFromPath(v2rayPath)
	if fromInfo == Xray || (fromInfo == "" && fromName == Xray) {
		return rememberVersion(v2rayPath, Xray, "", nil)
	}

	variant, ver, err = readCoreVersion(v2rayPath)
	if err != nil {
		if errors.Is(err, ErrProbeKilled) {
			known := fromInfo
			if known == "" {
				known = fromName
			}
			return rememberVersion(v2rayPath, known, "", err)
		}
		return "", "", err
	}
	// The printed banner names the variant. Build info is the module that
	// was linked, so it wins when the banner and the module disagree.
	if fromInfo == V2rayaCore || fromInfo == V2ray {
		variant = fromInfo
	}
	return rememberVersion(v2rayPath, variant, ver, nil)
}

func rememberVersion(path string, variant Variant, ver string, err error) (Variant, string, error) {
	v2rayVersion.version = ver
	v2rayVersion.variant = variant
	v2rayVersion.binPath = path
	v2rayVersion.err = err
	v2rayVersion.lastUpdate = time.Now()
	return variant, ver, err
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

func variantFromPath(path string) Variant {
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(path)), ".exe")
	switch base {
	case "xray":
		return Xray
	case "v2ray":
		return V2ray
	case "v2raya_core":
		return V2rayaCore
	default:
		return ""
	}
}

// variantFromBuildInfo reads the Go build info already stored in the binary.
// It does not run the program. Main.Path is the module that was linked, so an
// xray binary is not confused with v2raya_core, which only depends on xray.
func variantFromBuildInfo(path string) Variant {
	info, err := buildinfo.ReadFile(path)
	if err != nil || info == nil {
		return ""
	}
	mainPath := info.Main.Path
	if mainPath == "" {
		mainPath = info.Path
	}
	switch {
	case strings.Contains(mainPath, "github.com/xtls/xray-core"):
		return Xray
	case strings.Contains(mainPath, "github.com/v2fly/v2ray-core"):
		return V2ray
	case strings.Contains(mainPath, "github.com/v2rayA/v2raya-core"):
		return V2rayaCore
	default:
		return ""
	}
}

func readCoreVersion(path string) (variant Variant, ver string, err error) {
	out, killed, runErr := execCore(path, "version")
	if variant, ver, err = parseCoreVersion(out); err == nil {
		return variant, ver, nil
	}
	// A binary that does not exit (the version argument was ignored) is not
	// asked again. v2ray 4.x exits from an unknown `version` subcommand and
	// answers `-version`. A killed probe is not asked again either: the
	// second start is what runs the router out of memory.
	if !killed {
		out, killed, runErr = execCore(path, "-version")
		if variant, ver, err = parseCoreVersion(out); err == nil {
			return variant, ver, nil
		}
	}
	if killed {
		if runErr == nil {
			runErr = context.DeadlineExceeded
		}
		return "", "", fmt.Errorf("%w: %v", ErrProbeKilled, runErr)
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
	killed = errors.Is(ctx.Err(), context.DeadlineExceeded) || commandSignaled(err)
	return buf.String(), killed, err
}

type processSignaled interface {
	Signaled() bool
}

func commandSignaled(err error) bool {
	if err == nil {
		return false
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(processSignaled); ok && status.Signaled() {
			return true
		}
	}
	return strings.Contains(err.Error(), "signal: killed")
}
