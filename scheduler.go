package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// scheduledFeature 是可扩展的定时功能描述；以后新增功能 2、3 时只需增加配置项。
type scheduledFeature struct {
	name       string
	executable string
	arguments  []string
}

// scheduledFeatures 统一登记所有整点/半点执行的功能，当前先接入功能 1。
var scheduledFeatures = []scheduledFeature{
	{
		name:       "功能 1：同步每日统计到 emlog",
		executable: "/usr/local/bin/python3",
		arguments:  []string{userHomePath("studio", "codex-daily-stats", "sync_to_emlog.py")},
	},
}

var featureLogPath = userHomePath("studio", "codex-daily-stats", "go-hello-feature.log")
var authGoogleLogPath = userHomePath("studio", "authGoogle", "log.txt")

var featureLogMu sync.Mutex
var featureRunMu sync.Mutex

// startFeatureScheduler 在后台等待下一个整点或半点，不阻塞 Cocoa GUI 事件循环。
func startFeatureScheduler() {
	go watchImmediateFeatureRequests()
	go watchTodayNotificationCount()
	go func() {
		for {
			nextRun := nextHalfHour(time.Now())
			publishFeatureStatus(
				fmt.Sprintf("状态：运行中\n下次执行：%s", nextRun.Format("2006-01-02 15:04:05")),
				"等待定时执行。",
			)
			timer := time.NewTimer(time.Until(nextRun))
			<-timer.C
			if featureSchedulerIsPaused() {
				publishFeatureStatus(
					"状态：已暂停\n本次整点/半点已跳过",
					fmt.Sprintf("计划执行时间：%s\n功能 1 当前已暂停，未启动脚本。", nextRun.Format("2006-01-02 15:04:05")),
				)
				continue
			}
			publishFeatureStatus(
				fmt.Sprintf("状态：执行中\n开始时间：%s", nextRun.Format("2006-01-02 15:04:05")),
				"正在执行功能 1，请稍候……",
			)
			runScheduledFeatures(nextRun)
		}
	}()
}

// watchTodayNotificationCount 每 10 秒刷新一次菜单栏中的今日通知数量。
func watchTodayNotificationCount() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		refreshTodayNotificationCount()
	}
}

// refreshTodayNotificationCount 读取今日日期对应的日志行，并更新顶部菜单文字。
func refreshTodayNotificationCount() {
	count, err := countTodayNotificationLogs(authGoogleLogPath, time.Now())
	if err != nil {
		return
	}
	publishTodayNotificationCount(count)
}

// countTodayNotificationLogs 统计指定日志文件中属于给定日期的日志记录。
// 日志格式以 [YYYY-MM-DD HH:MM:SS] 开头，因此每条记录只需检查日期前缀。
func countTodayNotificationLogs(path string, day time.Time) (int, error) {
	logFile, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer logFile.Close()

	datePrefix := "[" + day.Format("2006-01-02")
	count := 0
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), datePrefix) {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

// watchImmediateFeatureRequests 轮询 GUI 的手动执行请求，不阻塞 Cocoa 主线程。
func watchImmediateFeatureRequests() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if featureSchedulerConsumeImmediateRequest() {
			runImmediateFeatures()
		}
	}
}

// runImmediateFeatures 立即执行全部已登记功能；手动执行不受自动暂停影响。
func runImmediateFeatures() {
	runAt := time.Now()
	publishFeatureStatus(
		fmt.Sprintf("状态：执行中\n手动开始：%s", runAt.Format("2006-01-02 15:04:05")),
		"正在手动执行功能 1，请稍候……",
	)
	runScheduledFeatures(runAt)
}

// nextHalfHour 计算严格晚于当前时刻的下一个整点/半点。
func nextHalfHour(now time.Time) time.Time {
	return now.Truncate(30 * time.Minute).Add(30 * time.Minute)
}

// runScheduledFeatures 按登记顺序执行全部定时功能，便于后续控制功能之间的顺序。
func runScheduledFeatures(runAt time.Time) {
	featureRunMu.Lock()
	defer featureRunMu.Unlock()

	for _, feature := range scheduledFeatures {
		runFeature(feature, runAt)
	}
}

// runFeature 执行单个外部功能，并把成功、失败和脚本输出写入日志。
func runFeature(feature scheduledFeature, runAt time.Time) {
	command := exec.Command(feature.executable, feature.arguments...)
	output, err := command.CombinedOutput()
	status := "成功"
	if err != nil {
		status = fmt.Sprintf("失败：%v", err)
	}

	detail := fmt.Sprintf("执行时间：%s\n命令：%s %s\n结果：%s\n输出：\n%s",
		runAt.Format("2006-01-02 15:04:05"), feature.executable, feature.arguments, status, string(output))
	publishFeatureStatus(fmt.Sprintf("状态：%s\n最近执行：%s", status, runAt.Format("2006-01-02 15:04:05")), detail)

	logLine := fmt.Sprintf("[%s] %s：%s\n输出：\n%s\n",
		runAt.Format("2006-01-02 15:04:05"), feature.name, status, string(output))
	writeFeatureLog(logLine)
}

// writeFeatureLog 追加功能执行记录，方便确认整点/半点任务是否实际运行。
func writeFeatureLog(line string) {
	featureLogMu.Lock()
	defer featureLogMu.Unlock()

	logFile, err := os.OpenFile(featureLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer logFile.Close()
	_, _ = logFile.WriteString(line)
}
