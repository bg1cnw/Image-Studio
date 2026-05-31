package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yuanhua/image-gptcodex/pkg/client"
)

const probeUpstreamTimeout = 20 * time.Second
const probeUpstreamMaxBody = 1 << 20

type ProbeResult struct {
	ModelCount int
}

func ProbeUpstream(ctx context.Context, cfg Config) (ProbeResult, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return ProbeResult{}, fmt.Errorf("API Key 不能为空")
	}
	baseURL, err := client.ValidateBaseURL(cfg.BaseURL)
	if err != nil {
		return ProbeResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, probeUpstreamTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("构造测活请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", client.UserAgent())

	proxyConfig, err := client.NormalizeProxyConfig(cfg.ProxyMode, cfg.ProxyURL)
	if err != nil {
		return ProbeResult{}, err
	}
	transport, err := client.NewHTTPTransport(proxyConfig)
	if err != nil {
		return ProbeResult{}, err
	}
	httpClient := &http.Client{Timeout: probeUpstreamTimeout, Transport: transport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("连接上游失败: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, probeUpstreamMaxBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		summary := summarizeProbeBody(body)
		if summary == "" && readErr != nil {
			summary = readErr.Error()
		}
		if summary != "" {
			return ProbeResult{}, fmt.Errorf("上游 /v1/models 返回 %d: %s", resp.StatusCode, summary)
		}
		return ProbeResult{}, fmt.Errorf("上游 /v1/models 返回 %d", resp.StatusCode)
	}
	if readErr != nil {
		return ProbeResult{}, fmt.Errorf("读取上游响应失败: %w", readErr)
	}

	var parsed struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ProbeResult{}, fmt.Errorf("上游 /v1/models 返回的 JSON 无效: %w", err)
	}
	if parsed.Data == nil {
		return ProbeResult{}, fmt.Errorf("上游 /v1/models 响应缺少 data 数组")
	}
	return ProbeResult{ModelCount: len(parsed.Data)}, nil
}

func summarizeProbeBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return ""
	}
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if msg := strings.TrimSpace(parsed.Error.Message); msg != "" {
			text = msg
		} else if msg := strings.TrimSpace(parsed.Message); msg != "" {
			text = msg
		}
	}
	if len(text) > 160 {
		return text[:160]
	}
	return text
}
