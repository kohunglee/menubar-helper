package main

import "testing"

// TestRemainingCodexQuota 确认接口的已使用百分比会正确转换为剩余百分比。
func TestRemainingCodexQuota(t *testing.T) {
	resetAt := int64(1730947200)
	quota := remainingCodexQuota(codexRateLimitSnapshot{
		Primary:   &codexRateLimitWindow{UsedPercent: 16, ResetsAt: &resetAt},
		Secondary: &codexRateLimitWindow{UsedPercent: 46},
	})
	if quota.FiveHour != 84 {
		t.Fatalf("5 小时剩余额度 = %d, want 84", quota.FiveHour)
	}
	if quota.Weekly != 54 {
		t.Fatalf("周剩余额度 = %d, want 54", quota.Weekly)
	}
	if quota.FiveHourResetAt != resetAt {
		t.Fatalf("5 小时重置时间 = %d, want %d", quota.FiveHourResetAt, resetAt)
	}
}

// TestRemainingCodexQuotaMissingWindow 确认缺失窗口不会伪造一个百分比。
func TestRemainingCodexQuotaMissingWindow(t *testing.T) {
	quota := remainingCodexQuota(codexRateLimitSnapshot{})
	if quota.FiveHour != -1 || quota.Weekly != -1 {
		t.Fatalf("缺失窗口应返回 -1，实际为 5 小时=%d、周=%d", quota.FiveHour, quota.Weekly)
	}
}

// TestQuotaForReadFailure 验证前 9 次失败保留旧数据，第 10 次失败才清空显示。
func TestQuotaForReadFailure(t *testing.T) {
	lastGood := codexQuota{FiveHour: 84, Weekly: 54, FiveHourResetAt: 1730947200}
	for failures := 1; failures < codexQuotaFailureDisplayLimit; failures++ {
		got := quotaForReadFailure(lastGood, true, failures)
		if got != lastGood {
			t.Fatalf("第 %d 次失败应保留旧数据，实际为 %+v", failures, got)
		}
	}
	cleared := quotaForReadFailure(lastGood, true, codexQuotaFailureDisplayLimit)
	if cleared.FiveHour != -1 || cleared.Weekly != -1 || cleared.FiveHourResetAt != 0 {
		t.Fatalf("第 10 次失败应清空显示，实际为 %+v", cleared)
	}
}
