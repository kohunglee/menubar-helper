package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Tink 是 macOS 自带的短促清脆提示音，适合作为截图完成反馈。
const screenshotSoundPath = "/System/Library/Sounds/Tink.aiff"

// screenshotSemaphore 防止截图命令尚未结束时，连续按键启动重复截图。
var screenshotSemaphore = make(chan struct{}, 1)

// startPrintScreenCapture 在后台监听 Print Screen 请求，不阻塞菜单栏和窗口事件循环。
func startPrintScreenCapture() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			if featurePrintScreenConsumeRequest() {
				triggerFullScreenCapture()
			}
		}
	}()
}

// triggerFullScreenCapture 异步执行截图，并丢弃截图进行中的重复触发。
func triggerFullScreenCapture() {
	select {
	case screenshotSemaphore <- struct{}{}:
		go func() {
			defer func() { <-screenshotSemaphore }()
			captureFullScreen()
		}()
	default:
		// 上一张截图还在保存，忽略重复按键，避免生成一组重复文件。
	}
}

// captureFullScreen 按“保存成功 -> 播放声音 -> 写入剪贴板”的顺序处理截图。
func captureFullScreen() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		publishScreenshotStatus("状态：无法读取用户目录")
		return
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")
	saveEnabled := featureScreenshotSaveEnabled()
	clipboardEnabled := featureScreenshotClipboardEnabled()
	soundEnabled := featureScreenshotSoundEnabled()
	if !saveEnabled && !clipboardEnabled && !soundEnabled {
		publishScreenshotStatus("状态：三个开关均已关闭，未执行截图")
		return
	}
	if !featureScreenCaptureAccessGranted() {
		publishScreenshotStatus("状态：未获得屏幕录制授权，本次不弹窗、不执行截图")
		return
	}

	if saveEnabled {
		if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
			publishScreenshotStatus("状态：无法创建下载目录")
			return
		}
	}

	fileName := fmt.Sprintf("书禾宝-全屏截图-%s.png", time.Now().Format("20060102-150405.000000000"))
	outputPath := filepath.Join(os.TempDir(), fileName)
	if saveEnabled {
		outputPath = filepath.Join(downloadsDir, fileName)
	}
	if !saveEnabled {
		// 关闭保存时只使用临时文件，复制完成后移入废纸篓，避免留下用户看不见的文件。
		defer func() { _ = exec.Command("/usr/bin/trash", outputPath).Run() }()
	}

	command := exec.Command("/usr/sbin/screencapture", "-x", outputPath)
	runErr := command.Run()

	fileInfo, err := os.Stat(outputPath)
	if err != nil || fileInfo.IsDir() || fileInfo.Size() == 0 {
		if runErr != nil {
			publishScreenshotStatus("状态：截图失败，请检查屏幕录制权限")
			return
		}
		publishScreenshotStatus("状态：截图没有成功保存")
		return
	}
	// screencapture 偶尔会在文件已写入后返回非零码；只要最终 PNG 有内容，就继续后续动作。

	// 只有截图文件已经存在且非空时，才按开关播放提示音。
	if soundEnabled {
		_ = exec.Command("/usr/bin/afplay", screenshotSoundPath).Run()
	}

	clipboardResult := "剪贴板未启用"
	if clipboardEnabled {
		if copyImageToClipboard(outputPath) {
			clipboardResult = "已复制到剪贴板"
		} else {
			clipboardResult = "复制到剪贴板失败"
		}
	}

	saveResult := "图片未保存到下载目录"
	if saveEnabled {
		saveResult = "已保存到下载目录"
	}
	soundResult := "声音未启用"
	if soundEnabled {
		soundResult = "已播放提示音"
	}
	publishScreenshotStatus(fmt.Sprintf("状态：截图成功\n%s；%s；%s", saveResult, soundResult, clipboardResult))
}
