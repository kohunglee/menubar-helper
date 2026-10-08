package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	deepSeekHost        = "127.0.0.1"
	deepSeekDefaultPort = 17891
)

var deepSeekExecutable = userHomePath(".local", "bin", "dsh")

// deepSeekProcessState 保存由书禾宝启动的 DeepSeek Harness 进程，便于准确关闭它。
type deepSeekProcessState struct {
	cmd      *exec.Cmd
	done     chan error
	port     int
	stopping bool
}

var (
	deepSeekProcessMu sync.Mutex
	deepSeekProcess   *deepSeekProcessState
	deepSeekPortValue = deepSeekDefaultPort
)

// startDeepSeekHarnessMonitor 轮询 GUI 的四个操作请求，并把耗时的进程控制放到后台。
func startDeepSeekHarnessMonitor() {
	go func() {
		deepSeekPortValue = featureDeepSeekPort()
		for {
			if featureDeepSeekConsumePortChangeRequest() {
				deepSeekPortValue = featureDeepSeekPort()
				publishDeepSeekStatus(fmt.Sprintf("端口已设置为 %d\n请点击“启动”使用此端口", deepSeekPortValue))
			}
			if featureDeepSeekConsumeStartRequest() {
				startDeepSeekHarness(deepSeekPortValue)
			}
			if featureDeepSeekConsumeOpenRequest() {
				openDeepSeekHarness(deepSeekPortValue)
			}
			if featureDeepSeekConsumeCloseRequest() {
				stopDeepSeekHarness(deepSeekPortValue)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
}

// startDeepSeekHarness 启动 dsh web，并避免重复启动相同的本地服务。
func startDeepSeekHarness(port int) {
	if port < 1 || port > 65535 {
		publishDeepSeekStatus("状态：端口必须是 1～65535 之间的数字")
		return
	}

	deepSeekProcessMu.Lock()
	if deepSeekProcess != nil {
		runningPort := deepSeekProcess.port
		deepSeekProcessMu.Unlock()
		publishDeepSeekStatus(fmt.Sprintf("状态：DeepSeek Harness 已在端口 %d 运行", runningPort))
		return
	}
	deepSeekProcessMu.Unlock()

	if pids := findDeepSeekPIDs(port); len(pids) > 0 {
		publishDeepSeekStatus(fmt.Sprintf("状态：DeepSeek Harness 已在端口 %d 运行", port))
		return
	}

	command := exec.Command(deepSeekExecutable, "web", "--host", deepSeekHost, "--port", strconv.Itoa(port))
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		publishDeepSeekStatus(fmt.Sprintf("状态：启动失败\n%s", err))
		return
	}

	state := &deepSeekProcessState{
		cmd:  command,
		done: make(chan error, 1),
		port: port,
	}
	deepSeekProcessMu.Lock()
	deepSeekProcess = state
	deepSeekProcessMu.Unlock()
	publishDeepSeekStatus(fmt.Sprintf("状态：正在启动\nhttp://%s:%d", deepSeekHost, port))

	// Wait 必须在后台消费，否则子进程退出后会留下僵尸进程。
	go waitDeepSeekHarness(state)
	go waitForDeepSeekReady(state)
}

// waitDeepSeekHarness 回收子进程并在意外退出时更新页面状态。
func waitDeepSeekHarness(state *deepSeekProcessState) {
	err := state.cmd.Wait()
	deepSeekProcessMu.Lock()
	if deepSeekProcess == state {
		deepSeekProcess = nil
	}
	stopping := state.stopping
	deepSeekProcessMu.Unlock()
	state.done <- err
	if stopping {
		return
	}
	if err != nil {
		publishDeepSeekStatus(fmt.Sprintf("状态：DeepSeek Harness 已退出\n%s", err))
		return
	}
	publishDeepSeekStatus("状态：DeepSeek Harness 已退出")
}

// waitForDeepSeekReady 轮询本地网页，只有服务真的能访问时才显示“运行中”。
func waitForDeepSeekReady(state *deepSeekProcessState) {
	url := deepSeekURL(state.port)
	client := &http.Client{Timeout: time.Second}
	deadline := time.NewTimer(15 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-state.done:
			return
		case <-deadline.C:
			publishDeepSeekStatus(fmt.Sprintf("状态：启动命令仍在运行，但网页暂未响应\n%s", url))
			return
		case <-ticker.C:
			request, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				continue
			}
			response, err := client.Do(request)
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode < http.StatusInternalServerError {
					publishDeepSeekStatus(fmt.Sprintf("状态：运行中\n%s", url))
					return
				}
			}
		}
	}
}

// openDeepSeekHarness 用系统默认浏览器打开当前端口，不负责启动服务。
func openDeepSeekHarness(port int) {
	url := deepSeekURL(port)
	if err := exec.Command("open", url).Start(); err != nil {
		publishDeepSeekStatus(fmt.Sprintf("状态：打开浏览器失败\n%s", err))
		return
	}
	publishDeepSeekStatus(fmt.Sprintf("状态：已打开浏览器\n%s", url))
}

// stopDeepSeekHarness 先优雅终止书禾宝启动的进程，也兼容关闭同端口的外部 dsh 进程。
func stopDeepSeekHarness(port int) {
	deepSeekProcessMu.Lock()
	state := deepSeekProcess
	if state != nil {
		state.stopping = true
	}
	deepSeekProcessMu.Unlock()

	if state != nil && state.cmd.Process != nil {
		_ = state.cmd.Process.Signal(os.Interrupt)
	}

	for _, pid := range findDeepSeekPIDs(port) {
		if state != nil && state.cmd.Process != nil && pid == state.cmd.Process.Pid {
			continue
		}
		if process, err := os.FindProcess(pid); err == nil {
			_ = process.Signal(os.Interrupt)
		}
	}

	if state != nil {
		select {
		case <-state.done:
		case <-time.After(2 * time.Second):
			if state.cmd.Process != nil {
				_ = state.cmd.Process.Kill()
			}
			<-state.done
		}
	}
	publishDeepSeekStatus(fmt.Sprintf("状态：已关闭 DeepSeek Harness（端口 %d）", port))
}

// findDeepSeekPIDs 只匹配 dsh 的完整命令行，避免误关其他本地程序。
func findDeepSeekPIDs(port int) []int {
	pattern := "^" + regexp.QuoteMeta(deepSeekExecutable+" web --host "+deepSeekHost+" --port "+strconv.Itoa(port)) + "$"
	output, err := exec.Command("pgrep", "-f", pattern).Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, value := range strings.Fields(string(output)) {
		pid, err := strconv.Atoi(value)
		if err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

// deepSeekURL 统一生成当前端口的本地访问地址。
func deepSeekURL(port int) string {
	return fmt.Sprintf("http://%s:%d", deepSeekHost, port)
}
