package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

var retryableMarkers = []string{
	"error code 524",
	"524: a timeout occurred",
	"error code 504",
	"gateway time-out",
	"service temporarily unavailable",
	"origin_gateway_timeout",
}

// IsRetryable mirrors Python is_retryable_response.
func IsRetryable(raw string) bool {
	text := strings.TrimSpace(raw)
	lower := strings.ToLower(text)
	for _, m := range retryableMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return false
	}
	if v, ok := data["retryable"].(bool); ok && v {
		return true
	}
	if status, ok := data["status"].(float64); ok {
		switch int(status) {
		case 502, 503, 504, 524:
			return true
		}
	}
	if errObj, ok := data["error"].(map[string]any); ok {
		message, _ := errObj["message"].(string)
		errType, _ := errObj["type"].(string)
		if strings.Contains(strings.ToLower(message), "temporarily unavailable") {
			return true
		}
		switch strings.ToLower(errType) {
		case "api_error", "server_error":
			return true
		}
	}
	return false
}

// DescribeProblem returns a human-readable Chinese explanation of an
// upstream failure body. Mirrors Python describe_response_problem.
func DescribeProblem(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "接口返回为空。"
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "error code 524") || strings.Contains(lower, "524: a timeout occurred") {
		return "Cloudflare 524:源站在超时时间内没有返回有效响应。"
	}
	if strings.Contains(lower, "error code 504") || strings.Contains(lower, "gateway time-out") {
		return "Cloudflare 504:源站网关超时。"
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err == nil && data != nil {
		var statusLabel string
		if status, ok := data["status"].(float64); ok {
			statusLabel = fmt.Sprintf("%d", int(status))
		}
		if name, ok := data["error_name"].(string); ok && (name == "origin_gateway_timeout" || name == "timeout") {
			if statusLabel == "" {
				statusLabel = name
			}
		}
		if statusLabel != "" {
			return fmt.Sprintf("接口返回 %s:上游服务超时。", statusLabel)
		}
		if errObj, ok := data["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok && msg != "" {
				return fmt.Sprintf("接口返回错误:%s", msg)
			}
			b, _ := json.Marshal(errObj)
			return fmt.Sprintf("接口返回错误:%s", string(b))
		}
		if msg, ok := data["message"].(string); ok && msg != "" {
			return fmt.Sprintf("接口返回消息:%s", msg)
		}
	}

	for ev := range IterEvents(raw) {
		if resp, ok := ev["response"].(map[string]any); ok {
			if errObj, ok := resp["error"]; ok && errObj != nil {
				b, _ := json.Marshal(errObj)
				return fmt.Sprintf("接口返回错误:%s", string(b))
			}
		}
		if errObj, ok := ev["error"]; ok && errObj != nil {
			b, _ := json.Marshal(errObj)
			return fmt.Sprintf("接口返回错误:%s", string(b))
		}
	}

	return "接口已返回内容,但没有发现 image_generation_call.result。"
}
