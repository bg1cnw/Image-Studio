package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// RequestAndExtract performs one HTTP request (no retry) and returns the parsed image.
// It writes the raw response stream to rawSink, and (optionally) reports progress
// via the supplied callback. The callback is invoked from the goroutine that calls
// RequestAndExtract; receivers should be cheap or buffer internally.
func RequestAndExtract(
	ctx context.Context,
	transport Transport,
	opts Options,
	rawSink io.Writer,
	onProgress func(stage string, elapsedSeconds int, bytesReceived int64),
) (ImageResult, error) {
	payload, err := BuildPayload(opts)
	if err != nil {
		return ImageResult{}, err
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}
	req := Request{
		URL:     baseURL + "/v1/responses",
		APIKey:  opts.APIKey,
		Payload: payload,
	}

	// Capture raw response in memory AND forward to rawSink so we can parse it
	// even when the caller only kept it as a file on disk.
	var buf bytes.Buffer
	tee := io.MultiWriter(rawSink, &buf)

	progressCh := make(chan string, 16)
	done := make(chan error, 1)
	startedAt := time.Now()

	go func() {
		done <- transport.Stream(ctx, req, tee, progressCh)
		close(progressCh)
	}()

	ticker := time.NewTicker(time.Duration(StatusIntervalSecond) * time.Second)
	defer ticker.Stop()

	lastStage := "等待接口响应"
	var streamErr error
loop:
	for {
		select {
		case <-ctx.Done():
			// Wait for goroutine to wind down so we don't leak.
			<-done
			return ImageResult{}, ctx.Err()
		case err, ok := <-done:
			if ok {
				streamErr = err
			}
			break loop
		case stage, ok := <-progressCh:
			if !ok {
				// Channel closed before done signal — drain.
				continue
			}
			lastStage = stage
		case <-ticker.C:
			if onProgress != nil {
				elapsed := int(time.Since(startedAt).Seconds())
				onProgress(lastStage, elapsed, int64(buf.Len()))
			}
		}
	}

	if streamErr != nil {
		// Even on transport failure, the buffer may contain a partial body
		// that the caller's retry logic wants to inspect — but parsing here
		// is not the job, just bubble up.
		return ImageResult{}, streamErr
	}

	rawText := buf.String()
	result, err := ExtractImageResult(rawText)
	if err != nil {
		return ImageResult{}, err
	}
	return result, nil
}

// RequestAndExtractWithRetries wraps RequestAndExtract with the same retry
// policy as the Python script. It writes one raw-response file per attempt
// (gptcodex-response-{timestamp}-attempt{N}.txt) under outputDir.
//
// Returns the final ImageResult and the path of the last raw-response file
// (handy for the CLI to print).
func RequestAndExtractWithRetries(
	ctx context.Context,
	transport Transport,
	opts Options,
	outputDir string,
	timestamp string,
	onLog func(string),
	onProgress func(stage string, elapsed int, bytes int64),
) (ImageResult, string, error) {
	if onLog == nil {
		onLog = func(string) {}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return ImageResult{}, "", fmt.Errorf("create output dir: %w", err)
	}

	var lastErr error
	var lastPath string

	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		rawPath := filepath.Join(outputDir, fmt.Sprintf("gptcodex-response-%s-attempt%d.txt", timestamp, attempt))
		lastPath = rawPath
		onLog(fmt.Sprintf("第 %d/%d 次请求...", attempt, MaxAttempts))

		f, err := os.Create(rawPath)
		if err != nil {
			return ImageResult{}, lastPath, fmt.Errorf("create raw response file: %w", err)
		}

		result, reqErr := RequestAndExtract(ctx, transport, opts, f, onProgress)
		f.Close()

		if reqErr == nil {
			return result, rawPath, nil
		}

		// Decide whether to retry based on the body we just wrote.
		rawBytes, _ := os.ReadFile(rawPath)
		raw := string(rawBytes)

		if errors.Is(reqErr, ErrNoImageInResponse) {
			lastErr = reqErr
			reason := DescribeProblem(raw)
			if attempt < MaxAttempts && IsRetryable(raw) {
				onLog(reason)
				onLog(fmt.Sprintf("这是可重试错误,%d 秒后自动重试...", RetryBackoffSeconds))
				if !sleepCtx(ctx, time.Duration(RetryBackoffSeconds)*time.Second) {
					return ImageResult{}, lastPath, ctx.Err()
				}
				continue
			}
			abs, _ := filepath.Abs(rawPath)
			return ImageResult{}, lastPath, fmt.Errorf("%s\n原始返回已保存:%s", reason, abs)
		}

		// Transport-level error (network, curl exit). Retry up to MaxAttempts.
		lastErr = reqErr
		if attempt < MaxAttempts {
			onLog(fmt.Sprintf("%v", reqErr))
			onLog(fmt.Sprintf("%d 秒后自动重试...", RetryBackoffSeconds))
			if !sleepCtx(ctx, time.Duration(RetryBackoffSeconds)*time.Second) {
				return ImageResult{}, lastPath, ctx.Err()
			}
			continue
		}
		return ImageResult{}, lastPath, reqErr
	}

	abs, _ := filepath.Abs(lastPath)
	if lastErr != nil {
		return ImageResult{}, lastPath, fmt.Errorf("多次请求后仍未成功。最后一次原始返回:%s: %w", abs, lastErr)
	}
	return ImageResult{}, lastPath, fmt.Errorf("多次请求后仍未成功。最后一次原始返回:%s", abs)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
