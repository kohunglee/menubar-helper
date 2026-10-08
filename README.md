# 书禾宝 App

这是一个最小的 macOS GUI 示例：Go 负责程序入口，Cocoa 负责原生窗口、按钮和菜单栏常驻。

关闭窗口左上角的红色按钮只会隐藏窗口，顶部栏的 Go 图标仍会保留。点击顶部栏图标后，可以选择“打开窗口”或“彻底退出”。

菜单栏下拉框顶部会显示“今日通知N”，其中 N 是 `~/studio/authGoogle/log.txt` 中当天日志记录数，程序每 10 秒自动刷新一次。

顶部栏会紧凑显示 `1.5 84 54 [图标] 28`；第一位表示 5 小时额度距离重置的小时数，保留一位小数，中间两位省略百分号以节省空间，图标右侧最后一位是今日通知数量。数据通过本机 Codex app-server 的 `account/rateLimits/read` 读取；打开“额度设置”分页，可以分别隐藏中间两个额度数字，并调整刷新间隔（默认 20 秒，允许 5～3600 秒）。

鼠标悬浮在顶部栏额度项目上时，第一行会显示 5 小时额度距离重置还有多少小时和分钟，第二行保留两个数字的含义说明。

额度接口短暂读取失败时，顶部栏保留上一次成功数据，并每 5 秒重试；连续失败 10 次后才显示 `-- -- --`。重新读取成功后立即恢复正常刷新间隔和新数据。

“DeepSeek”分页用于管理本机 DeepSeek Harness，默认命令为 `~/.local/bin/dsh web --host 127.0.0.1 --port 17891`，提供“启动”“open”“关闭”和“修改端口”四个操作。

“微微倒计时”分页可配置顶部菜单的三个分钟档位。未计时时菜单显示三个档位；开始后切换为剩余时间、独立的“暂停”“继续”和“停止倒计时”选项，停止后恢复三个档位。选择档位后出现约 48 像素、置顶可拖动的红色倒计时圆球，指针悬停时显示手形，分秒在球内居中；圆球右键可隐藏或隐藏一分钟，隐藏期间计时继续，也可从分页重新显示。结束时会在屏幕中央显示约 325 像素、置顶的标准“时间到”窗口；关闭窗口后本次计时重置，菜单恢复三个档位。

## Print Screen 全屏截图

打开书禾宝后，切换到“截图功能”分页，可以分别开启或关闭以下三个动作：

- 保存截图到 `~/Downloads`
- 复制截图到系统剪贴板
- 截图成功后播放清脆提示音

三个开关互不影响。按下外接键盘的 Print Screen 键后，程序会先确认 PNG 已成功落盘，再按顺序播放声音、复制到剪贴板；如果关闭了保存开关，程序会使用临时文件完成其他已开启动作并清理临时文件。macOS 通常会把外接键盘的 Print Screen 映射为 F13（keyCode 105）。

圆角版应用图标素材位于 `assets/app-icon-white-twintail-rounded.png`，最终 macOS 图标位于 `assets/AppIcon-white-twintail-rounded.icns`。

## 功能 1：整点和半点同步

App 常驻后台后，会在每个整点和半点执行一次：

```bash
/usr/local/bin/python3 ~/studio/codex-daily-stats/sync_to_emlog.py
```

执行记录会追加到 `~/studio/codex-daily-stats/go-hello-feature.log`。后续功能 2、3 可以继续添加到 `scheduler.go` 的 `scheduledFeatures` 清单中。

GUI 会显示功能 1 的运行状态、下次执行时间、最近一次执行详情和脚本输出。点击“暂停”会跳过后续整点/半点任务，点击“继续”恢复调度。

点击“立即执行”会立刻手动运行当前已登记的功能，并将执行时间、命令、结果和输出显示在 GUI 中；手动执行不受自动调度暂停影响。

## 构建

```bash
go build -o GoWhiteTwinTail .
```

应用包的图标素材为 `assets/AppIcon-white-twintail-rounded.icns`，顶部栏图标为 `assets/StatusIcon-white-twintail-rounded.png`。

## 语言选择

在“只要一个窗口和一个按钮”的需求里，Go 比 Rust 更省 Token 和时间：编译器、语法和单文件入口都更轻。若完全不限定 Go/Rust，Swift + SwiftUI 会是 macOS 原生 GUI 的最短路线；在 Go 和 Rust 两种语言中，这个例子选 Go。
