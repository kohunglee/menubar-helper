package main

import (
	"os"
	"testing"
	"time"
)

// TestNextHalfHour 确认调度器总是指向下一个整点或半点。
func TestNextHalfHour(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "整点之后",
			now:  time.Date(2026, time.August, 30, 5, 0, 1, 0, location),
			want: time.Date(2026, time.August, 30, 5, 30, 0, 0, location),
		},
		{
			name: "半点之后",
			now:  time.Date(2026, time.August, 30, 18, 30, 1, 0, location),
			want: time.Date(2026, time.August, 30, 19, 0, 0, 0, location),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := nextHalfHour(test.now); !got.Equal(test.want) {
				t.Fatalf("nextHalfHour() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestCountTodayNotificationLogs 确认只统计目标日期的日志记录。
func TestCountTodayNotificationLogs(t *testing.T) {
	logPath := t.TempDir() + "/log.txt"
	content := "[2026-08-31 09:00:00] 第一条\n输出：ok\n[2026-08-31 10:00:00] 第二条\n[2026-08-30 23:59:00] 昨日记录\n"
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("写入测试日志失败：%v", err)
	}

	day := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.Local)
	got, err := countTodayNotificationLogs(logPath, day)
	if err != nil {
		t.Fatalf("countTodayNotificationLogs() error = %v", err)
	}
	if got != 2 {
		t.Fatalf("countTodayNotificationLogs() = %d, want 2", got)
	}
}
