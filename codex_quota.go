package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// codexRateLimitWindow 对应 Codex app-server 返回的一个额度窗口。
type codexRateLimitWindow struct {
	UsedPercent int    `json:"usedPercent"`
	ResetsAt    *int64 `json:"resetsAt"`
}

// codexRateLimitSnapshot 对应 Codex 当前账号的 5 小时和周额度快照。
type codexRateLimitSnapshot struct {
	Primary   *codexRateLimitWindow `json:"primary"`
	Secondary *codexRateLimitWindow `json:"secondary"`
}

// codexRateLimitResult 是 account/rateLimits/read 的响应主体。
type codexRateLimitResult struct {
	RateLimits codexRateLimitSnapshot `json:"rateLimits"`
}

// codexRateLimitResponse 只保留读取额度所需的 JSON-RPC 字段。
type codexRateLimitResponse struct {
	ID     json.RawMessage       `json:"id"`
	Result *codexRateLimitResult `json:"result"`
	Error  *json.RawMessage      `json:"error"`
}

// codexQuota 表示书禾宝顶部栏要展示的剩余额度百分比和 5 小时窗口重置时间。
type codexQuota struct {
	FiveHour        int
	Weekly          int
	FiveHourResetAt int64
}

const (
	codexAppServerPath            = "/Applications/ChatGPT.app/Contents/Resources/codex"
	codexQuotaReadRetrySeconds    = 5
	codexQuotaFailureDisplayLimit = 10
)

var (
	codexQuotaLastGood          codexQuota
	codexQuotaHasLastGood       bool
	codexQuotaConsecutiveErrors int
)

// startCodexQuotaMonitor 定时读取 Codex 额度，并按照 GUI 设置的秒数刷新。
func startCodexQuotaMonitor() {
	go func() {
		// 给 Cocoa 窗口和菜单栏留出初始化时间，避免第一次数据更新被丢弃。
		time.Sleep(time.Second)
		for {
			readSucceeded := refreshCodexQuota()
			seconds := codexQuotaReadRetrySeconds
			if readSucceeded {
				seconds = featureQuotaRefreshIntervalSeconds()
			}
			if seconds < 5 {
				seconds = 5
			}
			time.Sleep(time.Duration(seconds) * time.Second)
		}
	}()
}

// quotaForReadFailure 在短暂读取失败时保留最近一次成功数据，连续失败达到阈值后才清空显示。
func quotaForReadFailure(lastGood codexQuota, hasLastGood bool, consecutiveFailures int) codexQuota {
	if hasLastGood && consecutiveFailures < codexQuotaFailureDisplayLimit {
		return lastGood
	}
	return codexQuota{FiveHour: -1, Weekly: -1}
}

// refreshCodexQuota 读取一次额度；失败时保留旧数据，并返回本次是否读取成功。
func refreshCodexQuota() bool {
	quota, err := readCodexQuota()
	if err != nil {
		codexQuotaConsecutiveErrors++
		fallbackQuota := quotaForReadFailure(codexQuotaLastGood, codexQuotaHasLastGood, codexQuotaConsecutiveErrors)
		publishCodexQuota(fallbackQuota.FiveHour, fallbackQuota.Weekly, fallbackQuota.FiveHourResetAt, "状态：暂时无法读取 Codex 额度")
		return false
	}
	codexQuotaConsecutiveErrors = 0
	codexQuotaLastGood = quota
	codexQuotaHasLastGood = true
	publishCodexQuota(quota.FiveHour, quota.Weekly, quota.FiveHourResetAt, fmt.Sprintf("状态：已更新，%s", time.Now().Format("2006-01-02 15:04:05")))
	return true
}

// readCodexQuota 通过本机 Codex app-server 读取真实账号额度，不读取网页或日志猜测。
func readCodexQuota() (codexQuota, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, codexAppServerPath, "app-server", "--stdio")
	command.Stderr = io.Discard
	stdout, err := command.StdoutPipe()
	if err != nil {
		return codexQuota{}, err
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		return codexQuota{}, err
	}
	if err := command.Start(); err != nil {
		return codexQuota{}, err
	}

	// 不立即关闭 stdin：Codex 服务需要一点时间完成网络额度查询。
	requests := []string{
		`{"method":"initialize","id":1,"params":{"clientInfo":{"name":"shuhebao","title":"书禾宝","version":"1.0"},"capabilities":{}}}`,
		`{"method":"initialized","params":{}}`,
		`{"method":"account/rateLimits/read","id":2,"params":null}`,
	}
	for _, request := range requests {
		if _, err := io.WriteString(stdin, request+"\n"); err != nil {
			_ = stdin.Close()
			_ = command.Wait()
			return codexQuota{}, err
		}
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var response codexRateLimitResponse
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			continue
		}
		if string(response.ID) != "2" {
			continue
		}
		_ = stdin.Close()
		if response.Error != nil {
			_ = command.Wait()
			return codexQuota{}, fmt.Errorf("Codex 额度接口返回错误")
		}
		if response.Result == nil {
			_ = command.Wait()
			return codexQuota{}, fmt.Errorf("Codex 额度接口没有返回数据")
		}
		_ = command.Wait()
		return remainingCodexQuota(response.Result.RateLimits), nil
	}

	_ = stdin.Close()
	_ = command.Wait()
	if err := scanner.Err(); err != nil {
		return codexQuota{}, err
	}
	return codexQuota{}, fmt.Errorf("Codex 额度接口未返回结果")
}

// remainingCodexQuota 将接口返回的 usedPercent 转为剩余百分比。
func remainingCodexQuota(snapshot codexRateLimitSnapshot) codexQuota {
	quota := codexQuota{FiveHour: -1, Weekly: -1}
	if snapshot.Primary != nil && snapshot.Primary.UsedPercent >= 0 && snapshot.Primary.UsedPercent <= 100 {
		quota.FiveHour = 100 - snapshot.Primary.UsedPercent
	}
	if snapshot.Primary != nil && snapshot.Primary.ResetsAt != nil && *snapshot.Primary.ResetsAt > 0 {
		quota.FiveHourResetAt = *snapshot.Primary.ResetsAt
	}
	if snapshot.Secondary != nil && snapshot.Secondary.UsedPercent >= 0 && snapshot.Secondary.UsedPercent <= 100 {
		quota.Weekly = 100 - snapshot.Secondary.UsedPercent
	}
	return quota
}
