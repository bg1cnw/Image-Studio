package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

// Transport is the abstraction over HTTP-with-SSE used by the client.
// Stream MUST write the raw response body (line-by-line) to rawSink,
// and SHOULD push human-readable status updates onto progress (best-effort).
// The progress channel is owned by the caller; Transport does not close it.
type Transport interface {
	Stream(ctx context.Context, req Request, rawSink io.Writer, progress chan<- string) error
}

// PickTransport returns a concrete Transport given the user's choice.
// "auto" prefers native, with the curl binary used only when explicitly asked.
func PickTransport(kind TransportKind) (Transport, error) {
	switch kind {
	case "", TransportAuto, TransportNative:
		return &NativeTransport{}, nil
	case TransportCurl:
		bin, err := locateCurl()
		if err != nil {
			return nil, err
		}
		return &CurlTransport{Binary: bin}, nil
	default:
		return nil, fmt.Errorf("unknown transport: %s", kind)
	}
}

// locateCurl finds the curl binary or returns an error.
func locateCurl() (string, error) {
	candidates := []string{"curl"}
	if runtime.GOOS == "windows" {
		candidates = []string{"curl.exe", "curl"}
	}
	for _, name := range candidates {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	// Last-ditch: System32 on Windows ships curl since Win10 1803.
	if runtime.GOOS == "windows" {
		sys := os.Getenv("SystemRoot")
		if sys == "" {
			sys = `C:\Windows`
		}
		candidate := sys + `\System32\curl.exe`
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("找不到 curl 可执行文件,无法启用 curl fallback")
}
