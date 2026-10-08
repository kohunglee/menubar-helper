package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Carbon -framework CoreGraphics -framework CoreText

#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>
#import <CoreGraphics/CoreGraphics.h>
#import <CoreText/CoreText.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>

// 全局持有控制器，确保窗口隐藏后菜单栏图标仍然可用。
static id helloController;
static volatile BOOL featureSchedulerPaused = NO;
static volatile BOOL featureRunNowRequested = NO;
static volatile int todayNotificationCount = 0;
static volatile BOOL printScreenRequested = NO;
static volatile BOOL screenshotSaveEnabled = YES;
static volatile BOOL screenshotClipboardEnabled = YES;
static volatile BOOL screenshotSoundEnabled = YES;
static volatile BOOL quotaFiveHourVisible = YES;
static volatile BOOL quotaWeeklyVisible = YES;
static volatile int quotaRefreshSeconds = 20;
static volatile int quotaFiveHourValue = -1;
static volatile int quotaWeeklyValue = -1;
static volatile long long quotaFiveHourResetAt = 0;
static volatile BOOL deepSeekStartRequested = NO;
static volatile BOOL deepSeekOpenRequested = NO;
static volatile BOOL deepSeekCloseRequested = NO;
static volatile BOOL deepSeekPortChangeRequested = NO;
static volatile int deepSeekPort = 17891;
static EventHotKeyRef printScreenHotKeyRef = NULL;
static EventHandlerRef printScreenEventHandlerRef = NULL;
static int timerMinutes[3] = {10, 20, 30};
static NSInteger timerRemainingSeconds = 0;
static BOOL timerRunning = NO;
static BOOL timerPaused = NO;
static BOOL timerSessionActive = NO;
static BOOL timerHidden = NO;
static NSDate *timerHiddenUntil = nil;
static NSTimer *timerTick = nil;
static NSPanel *timerBallPanel = nil;
static NSView *timerBallView = nil;
static NSWindow *timerDoneWindow = nil;

// 红色倒计时浮球，右键提供隐藏操作。
@interface TimerBallView : NSView
@property(nonatomic, assign) id owner;
@property(nonatomic, assign) NSPoint dragStart;
@property(nonatomic, assign) NSPoint windowStart;
@property(nonatomic, copy) NSString *timerText;
@property(nonatomic, retain) NSTrackingArea *cursorTrackingArea;
@end
@implementation TimerBallView
- (void)updateTrackingAreas {
    if (self.cursorTrackingArea != nil) [self removeTrackingArea:self.cursorTrackingArea];
    self.cursorTrackingArea = [[NSTrackingArea alloc] initWithRect:self.bounds
                                                           options:NSTrackingMouseEnteredAndExited | NSTrackingMouseMoved | NSTrackingActiveAlways | NSTrackingInVisibleRect
                                                             owner:self
                                                          userInfo:nil];
    [self addTrackingArea:self.cursorTrackingArea];
    [super updateTrackingAreas];
}
- (void)mouseEntered:(NSEvent *)event { (void)event; [[NSCursor pointingHandCursor] set]; }
- (void)mouseMoved:(NSEvent *)event { (void)event; [[NSCursor pointingHandCursor] set]; }
- (void)cursorUpdate:(NSEvent *)event { (void)event; [[NSCursor pointingHandCursor] set]; }
- (void)mouseExited:(NSEvent *)event { (void)event; [[NSCursor arrowCursor] set]; }
- (void)resetCursorRects {
    [self addCursorRect:self.bounds cursor:[NSCursor pointingHandCursor]];
}
- (void)mouseDown:(NSEvent *)event { (void)event; self.dragStart = [NSEvent mouseLocation]; self.windowStart = self.window.frame.origin; }
- (void)mouseDragged:(NSEvent *)event {
    (void)event;
    NSPoint current = [NSEvent mouseLocation];
    [self.window setFrameOrigin:NSMakePoint(self.windowStart.x + current.x - self.dragStart.x,
                                            self.windowStart.y + current.y - self.dragStart.y)];
}
- (void)drawRect:(NSRect)dirtyRect {
    (void)dirtyRect;
    [[NSColor colorWithRed:0.88 green:0.12 blue:0.16 alpha:1.0] setFill];
    NSBezierPath *ballPath = [NSBezierPath bezierPathWithOvalInRect:NSInsetRect(self.bounds, 4, 4)];
    NSShadow *shadow = [[[NSShadow alloc] init] autorelease];
    [shadow setShadowColor:[[NSColor blackColor] colorWithAlphaComponent:0.24]];
    [shadow setShadowBlurRadius:5.0];
    [shadow setShadowOffset:NSMakeSize(0, -2)];
    [NSGraphicsContext saveGraphicsState];
    [shadow set];
    [ballPath fill];
    [NSGraphicsContext restoreGraphicsState];
    NSString *text = self.timerText ?: @"00:00";
    NSFont *font = [NSFont boldSystemFontOfSize:9];
    CTFontRef ctFont = CTFontCreateWithName((CFStringRef)font.fontName, font.pointSize, NULL);
    CGColorRef white = [[NSColor whiteColor] CGColor];
    const void *keys[] = {kCTFontAttributeName, kCTForegroundColorAttributeName};
    const void *values[] = {ctFont, white};
    CFDictionaryRef attributes = CFDictionaryCreate(kCFAllocatorDefault, keys, values, 2,
                                                    &kCFTypeDictionaryKeyCallBacks,
                                                    &kCFTypeDictionaryValueCallBacks);
    CFAttributedStringRef attributed = CFAttributedStringCreate(kCFAllocatorDefault, (CFStringRef)text, attributes);
    CTLineRef line = CTLineCreateWithAttributedString(attributed);
    CGFloat ascent = 0, descent = 0, leading = 0;
    double textWidth = CTLineGetTypographicBounds(line, &ascent, &descent, &leading);
    CGFloat x = floor((NSWidth(self.bounds) - textWidth) / 2.0);
    CGFloat baselineY = NSMidY(self.bounds) - (ascent - descent) / 2.0;
    CGContextRef context = [[NSGraphicsContext currentContext] CGContext];
    CGContextSetTextMatrix(context, CGAffineTransformIdentity);
    CGContextSetTextPosition(context, x, baselineY);
    CTLineDraw(line, context);
    CFRelease(line); CFRelease(attributed); CFRelease(attributes); CFRelease(ctFont);
}
- (void)rightMouseDown:(NSEvent *)event {
    (void)event;
    NSMenu *menu = [[NSMenu alloc] initWithTitle:@"微微倒计时"];
    NSMenuItem *hideMinute = [[NSMenuItem alloc] initWithTitle:@"隐藏一分钟" action:@selector(hideTimerBallOneMinute:) keyEquivalent:@""];
    [hideMinute setTarget:self.owner]; [menu addItem:hideMinute];
    NSMenuItem *hide = [[NSMenuItem alloc] initWithTitle:@"隐藏" action:@selector(hideTimerBall:) keyEquivalent:@""];
    [hide setTarget:self.owner]; [menu addItem:hide];
    [NSMenu popUpContextMenu:menu withEvent:event forView:self];
}
@end

// 时间到窗口忽略 Esc，只通过窗口按钮或窗口标题栏关闭按钮处理。
@interface TimerDoneWindow : NSWindow
@end
@implementation TimerDoneWindow
- (void)cancelOperation:(id)sender { (void)sender; }
@end

// 自定义状态栏视图需要主动把鼠标点击转交给 NSStatusItem，否则菜单不会弹出。
@interface ClickableStatusView : NSView
@property(nonatomic, assign) NSStatusItem *statusItem;
@end

@implementation ClickableStatusView
- (void)mouseDown:(NSEvent *)event {
    (void)event;
    [self.statusItem popUpStatusItemMenu:self.statusItem.menu];
}

- (void)rightMouseDown:(NSEvent *)event {
    (void)event;
    [self.statusItem popUpStatusItemMenu:self.statusItem.menu];
}
@end

// Carbon 热键回调在控制器定义前声明，供安装热键的方法使用。
static OSStatus handlePrintScreenHotKey(EventHandlerCallRef nextHandler,
                                        EventRef event,
                                        void *userData);

// 负责窗口显示、菜单栏图标、按钮点击和退出操作。
@interface HelloController : NSObject <NSWindowDelegate>
@property(nonatomic, retain) NSWindow *window;
@property(nonatomic, retain) NSStatusItem *statusItem;
@property(nonatomic, retain) NSTextField *featureStatusLabel;
@property(nonatomic, retain) NSTextView *featureDetailView;
@property(nonatomic, retain) NSButton *pauseButton;
@property(nonatomic, retain) NSButton *resumeButton;
@property(nonatomic, retain) NSButton *runNowButton;
@property(nonatomic, retain) NSMenuItem *todayNotificationItem;
@property(nonatomic, retain) NSTextField *screenshotStatusLabel;
@property(nonatomic, retain) NSButton *screenshotSaveToggle;
@property(nonatomic, retain) NSButton *screenshotClipboardToggle;
@property(nonatomic, retain) NSButton *screenshotSoundToggle;
@property(nonatomic, retain) NSTextField *fiveHourQuotaLabel;
@property(nonatomic, retain) NSTextField *weeklyQuotaLabel;
@property(nonatomic, retain) NSTextField *quotaStatusLabel;
@property(nonatomic, retain) NSButton *fiveHourQuotaToggle;
@property(nonatomic, retain) NSButton *weeklyQuotaToggle;
@property(nonatomic, retain) NSTextField *quotaRefreshSecondsField;
@property(nonatomic, retain) NSTimer *quotaTooltipTimer;
@property(nonatomic, retain) NSView *quotaStatusView;
@property(nonatomic, retain) NSTextField *quotaBarLabel;
@property(nonatomic, retain) NSTextField *notificationBarLabel;
@property(nonatomic, retain) NSTextField *deepSeekStatusLabel;
@property(nonatomic, retain) NSTextField *deepSeekPortField;
@property(nonatomic, retain) id globalKeyMonitor;
@property(nonatomic, retain) id localKeyMonitor;
@property(nonatomic, retain) NSMutableArray *timerMenuItems;
@property(nonatomic, retain) NSMenuItem *timerControlSeparator;
@property(nonatomic, retain) NSTextField *timerConfig10Field;
@property(nonatomic, retain) NSTextField *timerConfig20Field;
@property(nonatomic, retain) NSTextField *timerConfig30Field;
- (void)refreshCodexQuotaBarTitle;
@end

@implementation HelloController

// 显示主窗口，并把焦点切回这个小应用。
- (void)showWindow:(id)sender {
    (void)sender;
    [self.window makeKeyAndOrderFront:nil];
    [[NSApplication sharedApplication] activateIgnoringOtherApps:YES];
}

// 红色关闭按钮只隐藏窗口，不终止后台菜单栏进程。
- (BOOL)windowShouldClose:(NSWindow *)sender {
    if (sender == timerDoneWindow) {
        [self closeTimerDone:nil];
        return NO;
    }
    [sender orderOut:nil];
    return NO;
}

// 菜单栏图标中的“彻底退出”操作。
- (void)quitApp:(id)sender {
    (void)sender;
    [[NSApplication sharedApplication] terminate:nil];
}

// 保存三档分钟配置，并同步顶部菜单中的档位名称。
- (void)saveTimerConfiguration:(id)sender {
    (void)sender;
    NSTextField *fields[3] = {self.timerConfig10Field, self.timerConfig20Field, self.timerConfig30Field};
    for (int i = 0; i < 3; i++) {
        NSInteger value = [fields[i] integerValue];
        if (value < 1) value = 1;
        if (value > 999) value = 999;
        timerMinutes[i] = (int)value;
        [fields[i] setIntegerValue:value];
        [[NSUserDefaults standardUserDefaults] setInteger:value forKey:[NSString stringWithFormat:@"timerMinutes%d", i]];
        NSMenuItem *item = [self.timerMenuItems objectAtIndex:i];
        [item setTitle:[NSString stringWithFormat:@"%d 分钟", timerMinutes[i]]];
    }
}

// 根据菜单选择启动唯一倒计时，并创建屏幕上的红色圆球。
- (void)startTimerPreset:(id)sender {
    NSInteger index = [sender tag];
    if (index < 0 || index >= 3) return;
    timerRemainingSeconds = timerMinutes[index] * 60;
    timerRunning = YES; timerPaused = NO; timerSessionActive = YES; timerHidden = NO; timerHiddenUntil = nil;
    if (timerTick != nil) { [timerTick invalidate]; timerTick = nil; }
    // 普通 Timer 在菜单追踪事件模式下会暂停；加入 CommonModes 后菜单展开时仍每秒刷新剩余时间。
    timerTick = [NSTimer timerWithTimeInterval:1.0 target:self selector:@selector(timerSecond:) userInfo:nil repeats:YES];
    [[NSRunLoop mainRunLoop] addTimer:timerTick forMode:NSRunLoopCommonModes];
    if (timerBallPanel == nil) {
        timerBallPanel = [[NSPanel alloc] initWithContentRect:NSMakeRect(0, 0, 56, 56) styleMask:NSWindowStyleMaskBorderless backing:NSBackingStoreBuffered defer:NO];
        [timerBallPanel setLevel:NSStatusWindowLevel]; [timerBallPanel setHidesOnDeactivate:NO];
        [timerBallPanel setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorFullScreenAuxiliary];
        [timerBallPanel setOpaque:NO]; [timerBallPanel setBackgroundColor:[NSColor clearColor]];
        TimerBallView *ball = [[TimerBallView alloc] initWithFrame:NSMakeRect(0, 0, 56, 56)];
        ball.owner = self; ball.timerText = @"00:00"; timerBallView = ball;
        [timerBallPanel setContentView:ball];
        NSScreen *screen = [NSScreen mainScreen]; NSRect visible = [screen visibleFrame];
        [timerBallPanel setFrameOrigin:NSMakePoint(NSMaxX(visible) - 76, NSMaxY(visible) - 86)];
    }
    [timerBallPanel orderFrontRegardless]; [self refreshTimerDisplay];
    [self refreshTimerMenu];
}

// 更新倒计时圆球的分秒和自动恢复隐藏状态。
- (void)refreshTimerDisplay {
    NSInteger minutes = timerRemainingSeconds / 60, seconds = timerRemainingSeconds % 60;
    TimerBallView *ball = (TimerBallView *)timerBallView;
    ball.timerText = [NSString stringWithFormat:@"%02ld:%02ld", (long)minutes, (long)seconds];
    [ball setNeedsDisplay:YES];
    if (timerHidden && timerHiddenUntil != nil && [timerHiddenUntil timeIntervalSinceNow] <= 0) {
        timerHidden = NO; timerHiddenUntil = nil; [timerBallPanel orderFrontRegardless];
    }
}

// 每秒推进倒计时；暂停时不扣时，归零后显示可手动关闭的居中提示。
- (void)timerSecond:(NSTimer *)timer {
    (void)timer;
    if (!timerRunning || timerPaused) return;
    if (timerRemainingSeconds > 0) timerRemainingSeconds--;
    [self refreshTimerDisplay]; [self refreshTimerMenu];
    if (timerRemainingSeconds == 0) {
        timerRunning = NO; [timerTick invalidate]; timerTick = nil; [timerBallPanel orderOut:nil];
        if (timerDoneWindow == nil) {
            timerDoneWindow = [[TimerDoneWindow alloc] initWithContentRect:NSMakeRect(0, 0, 325, 325) styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable backing:NSBackingStoreBuffered defer:NO];
            [timerDoneWindow setTitle:@"微微倒计时"]; [timerDoneWindow setReleasedWhenClosed:NO]; [timerDoneWindow setDelegate:self];
            [timerDoneWindow setLevel:NSStatusWindowLevel];
            [timerDoneWindow setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorFullScreenAuxiliary];
            [timerDoneWindow setBackgroundColor:[NSColor colorWithCalibratedRed:0.94 green:0.94 blue:0.94 alpha:1]];
            NSTextField *done = [NSTextField labelWithString:@"时间到"]; [done setFrame:NSMakeRect(25, 135, 275, 60)];
            [done setFont:[NSFont boldSystemFontOfSize:40]]; [done setAlignment:NSTextAlignmentCenter]; [[timerDoneWindow contentView] addSubview:done];
            NSButton *close = [NSButton buttonWithTitle:@"关闭" target:self action:@selector(closeTimerDone:)];
            [close setFrame:NSMakeRect(105, 70, 115, 36)]; [close setBezelStyle:NSBezelStyleRounded]; [[timerDoneWindow contentView] addSubview:close];
        }
        [timerDoneWindow center];
        [timerDoneWindow makeKeyAndOrderFront:nil]; [[NSApplication sharedApplication] activateIgnoringOtherApps:YES];
        [self refreshTimerMenu];
    }
}

// 仅由“关闭”按钮收起时间到提示。
- (void)closeTimerDone:(id)sender {
    (void)sender;
    [timerDoneWindow orderOut:nil];
    timerRunning = NO; timerPaused = NO; timerSessionActive = NO; timerRemainingSeconds = 0;
    [self refreshTimerMenu];
}

// 通过菜单栏暂停、继续当前计时。
- (void)pauseTimer:(id)sender { (void)sender; if (!timerRunning || timerPaused) return; timerPaused = YES; [self refreshTimerMenu]; }
- (void)resumeTimer:(id)sender { (void)sender; if (!timerRunning || !timerPaused) return; timerPaused = NO; [self refreshTimerMenu]; }

// 停止倒计时并关闭浮球。
- (void)stopTimer:(id)sender {
    (void)sender; timerRunning = NO; timerPaused = NO; timerSessionActive = NO; timerRemainingSeconds = 0;
    if (timerTick != nil) { [timerTick invalidate]; timerTick = nil; }
    [timerBallPanel orderOut:nil]; [self refreshTimerMenu];
}

// 支持右键菜单中的立即隐藏与隐藏一分钟，隐藏期间时钟仍继续运行。
- (void)hideTimerBall:(id)sender { (void)sender; timerHidden = YES; timerHiddenUntil = nil; [timerBallPanel orderOut:nil]; }
- (void)hideTimerBallOneMinute:(id)sender { (void)sender; timerHidden = YES; timerHiddenUntil = [NSDate dateWithTimeIntervalSinceNow:60]; [timerBallPanel orderOut:nil]; }
- (void)showTimerBall:(id)sender { (void)sender; if (!timerRunning) return; timerHidden = NO; timerHiddenUntil = nil; [self refreshTimerDisplay]; [timerBallPanel orderFrontRegardless]; }

// 刷新菜单中的暂停/继续、停止可用状态及当前倒计时。
- (void)refreshTimerMenu {
    NSString *title = !timerSessionActive ? @"微微倒计时：未运行" : (timerRunning ? [NSString stringWithFormat:@"微微倒计时：%02ld:%02ld%@", (long)(timerRemainingSeconds/60), (long)(timerRemainingSeconds%60), timerPaused ? @"（已暂停）" : @""] : @"微微倒计时：时间到");
    for (int i = 0; i < 3; i++) [self.timerMenuItems[i] setHidden:timerSessionActive];
    [self.timerControlSeparator setHidden:!timerSessionActive];
    [self.timerMenuItems[3] setHidden:!timerSessionActive];
    [self.timerMenuItems[4] setHidden:!timerSessionActive];
    [self.timerMenuItems[5] setHidden:!timerSessionActive];
    [self.timerMenuItems[6] setHidden:!timerSessionActive];
    [self.timerMenuItems[3] setTitle:title];
    [self.timerMenuItems[4] setEnabled:timerRunning && !timerPaused];
    [self.timerMenuItems[5] setEnabled:timerRunning && timerPaused];
    [self.timerMenuItems[6] setEnabled:timerSessionActive];
}

// 暂停功能 1 的自动调度，并即时更新 GUI 上的按钮状态。
- (void)pauseFeature:(id)sender {
    (void)sender;
    featureSchedulerPaused = YES;
    [self.pauseButton setEnabled:NO];
    [self.resumeButton setEnabled:YES];
    [self.featureStatusLabel setStringValue:@"状态：已暂停\n整点/半点不会执行功能 1"];
}

// 恢复功能 1 的自动调度，并即时更新 GUI 上的按钮状态。
- (void)resumeFeature:(id)sender {
    (void)sender;
    featureSchedulerPaused = NO;
    [self.pauseButton setEnabled:YES];
    [self.resumeButton setEnabled:NO];
    [self.featureStatusLabel setStringValue:@"状态：运行中\n等待下一次整点/半点执行"];
}

// 发起一次手动执行；该操作不受自动调度暂停状态影响。
- (void)runFeatureNow:(id)sender {
    (void)sender;
    featureRunNowRequested = YES;
    [self.featureStatusLabel setStringValue:@"状态：准备立即执行\n正在等待后台调度器接收请求"];
    [self.featureDetailView setString:@"已收到手动执行请求，功能 1 即将启动……"];
}

// 接收后台调度器的状态和脚本输出，并更新 GUI 详情区域。
- (void)updateFeatureStatus:(NSString *)status detail:(NSString *)detail {
    if (status != nil && self.featureStatusLabel != nil) {
        [self.featureStatusLabel setStringValue:status];
    }
    if (detail != nil && self.featureDetailView != nil) {
        [self.featureDetailView setString:detail];
        [self.featureDetailView scrollRangeToVisible:NSMakeRange([[self.featureDetailView string] length], 0)];
    }
}

// 更新菜单栏顶部的今日通知数量，不给该菜单项绑定点击动作。
- (void)updateTodayNotificationCount:(int)count {
	 todayNotificationCount = count;
	 [self refreshCodexQuotaBarTitle];
    if (self.todayNotificationItem != nil) {
        NSString *title = [NSString stringWithFormat:@"今日通知%d", count];
        [self.todayNotificationItem setTitle:title];
    }
}

// 更新截图分页中的最近一次处理结果，方便确认三个开关是否生效。
- (void)updateScreenshotStatus:(NSString *)status {
    if (status != nil && self.screenshotStatusLabel != nil) {
        [self.screenshotStatusLabel setStringValue:status];
    }
}

// 接收 DeepSeek Harness 后台控制结果，并显示当前命令、端口或错误信息。
- (void)updateDeepSeekStatus:(NSString *)status {
    if (status != nil && self.deepSeekStatusLabel != nil) {
        [self.deepSeekStatusLabel setStringValue:status];
    }
}

// 切换是否把截图文件保存在下载目录；关闭时仍会临时生成图片供其他开关使用。
- (void)toggleScreenshotSave:(id)sender {
    screenshotSaveEnabled = [(NSButton *)sender state] == NSControlStateValueOn;
}

// 切换是否把已经成功生成的图片写入系统剪贴板。
- (void)toggleScreenshotClipboard:(id)sender {
    screenshotClipboardEnabled = [(NSButton *)sender state] == NSControlStateValueOn;
}

// 切换是否在截图成功后播放清脆提示音。
- (void)toggleScreenshotSound:(id)sender {
    screenshotSoundEnabled = [(NSButton *)sender state] == NSControlStateValueOn;
}

// 请求启动 DeepSeek Harness，由 Go 后台进程负责真正启动服务并反馈结果。
- (void)startDeepSeekHarness:(id)sender {
    (void)sender;
    deepSeekStartRequested = YES;
    [self.deepSeekStatusLabel setStringValue:@"状态：正在启动 DeepSeek Harness……"];
}

// 请求在默认浏览器中打开当前端口的 DeepSeek Harness 页面。
- (void)openDeepSeekHarness:(id)sender {
    (void)sender;
    deepSeekOpenRequested = YES;
    [self.deepSeekStatusLabel setStringValue:@"状态：正在打开浏览器……"];
}

// 请求关闭当前端口对应的 DeepSeek Harness 服务。
- (void)closeDeepSeekHarness:(id)sender {
    (void)sender;
    deepSeekCloseRequested = YES;
    [self.deepSeekStatusLabel setStringValue:@"状态：正在关闭 DeepSeek Harness……"];
}

// 校验并保存端口；如果服务正在运行，修改后需要重新点击“启动”才会使用新端口。
- (void)changeDeepSeekPort:(id)sender {
    (void)sender;
    NSInteger port = [self.deepSeekPortField integerValue];
    if (port < 1 || port > 65535) {
        [self.deepSeekStatusLabel setStringValue:@"状态：端口必须是 1～65535 之间的数字"];
        return;
    }
    deepSeekPort = (int)port;
    deepSeekPortChangeRequested = YES;
    [self.deepSeekStatusLabel setStringValue:[NSString stringWithFormat:@"端口已修改为 %ld\n请点击“启动”使用新端口", (long)port]];
}

// 根据真实重置时间刷新菜单栏悬浮提示，显示到小时和分钟。
- (void)refreshCodexQuotaTooltip:(id)sender {
    (void)sender;
    if (self.statusItem == nil) {
        return;
    }
    [self refreshCodexQuotaBarTitle];
    NSString *resetText = @"5小时额度重置时间暂不可用";
    if (quotaFiveHourResetAt > 0) {
        NSTimeInterval remainingSeconds = (NSTimeInterval)quotaFiveHourResetAt - [[NSDate date] timeIntervalSince1970];
        NSInteger remainingMinutes = remainingSeconds > 0 ? (NSInteger)floor(remainingSeconds / 60.0) : 0;
        NSInteger remainingHours = remainingMinutes / 60;
        NSInteger remainingMinutePart = remainingMinutes % 60;
        resetText = [NSString stringWithFormat:@"5小时额度还有 %ld 小时 %ld 分钟重置", (long)remainingHours, (long)remainingMinutePart];
    }
    [self.quotaStatusView setToolTip:[NSString stringWithFormat:@"%@\nCodex 额度：前一个为 5 小时，后一个为周额度", resetText]];
}

// 根据真实重置时间刷新菜单栏四个紧凑数字，最后一位是今日通知数量。
- (void)refreshCodexQuotaBarTitle {
    if (self.statusItem == nil) {
        return;
    }
    NSString *resetBarText = @"--";
    if (quotaFiveHourResetAt > 0) {
        NSTimeInterval remainingSeconds = (NSTimeInterval)quotaFiveHourResetAt - [[NSDate date] timeIntervalSince1970];
        double remainingHours = remainingSeconds > 0 ? remainingSeconds / 3600.0 : 0.0;
        resetBarText = [NSString stringWithFormat:@"%.1f", remainingHours];
    }
    NSString *fiveHourBarText = quotaFiveHourValue < 0 ? @"--" : [NSString stringWithFormat:@"%d", quotaFiveHourValue];
    NSString *weeklyBarText = quotaWeeklyValue < 0 ? @"--" : [NSString stringWithFormat:@"%d", quotaWeeklyValue];
    NSString *notificationBarText = [NSString stringWithFormat:@"%d", todayNotificationCount];
    NSString *barText = @"";
    if (quotaFiveHourVisible && quotaWeeklyVisible) {
        barText = [NSString stringWithFormat:@"%@ %@ %@", resetBarText, fiveHourBarText, weeklyBarText];
    } else if (quotaFiveHourVisible) {
        barText = [NSString stringWithFormat:@"%@ %@", resetBarText, fiveHourBarText];
    } else if (quotaWeeklyVisible) {
        barText = [NSString stringWithFormat:@"%@ %@", resetBarText, weeklyBarText];
    }
    [self.quotaBarLabel setStringValue:barText];
    [self.notificationBarLabel setStringValue:notificationBarText];
    [self.statusItem setVisible:YES];
}

// 更新顶部栏和额度设置页中的两个 Codex 剩余额度，并生成精确到分钟的重置提示。
- (void)updateCodexQuota:(int)fiveHour weekly:(int)weekly fiveHourResetAt:(long long)fiveHourResetAt status:(NSString *)status {
    NSString *fiveHourText = fiveHour < 0 ? @"5小时 --%" : [NSString stringWithFormat:@"5小时 %d%%", fiveHour];
    NSString *weeklyText = weekly < 0 ? @"周 --%" : [NSString stringWithFormat:@"周 %d%%", weekly];
    quotaFiveHourValue = fiveHour;
    quotaWeeklyValue = weekly;
    quotaFiveHourResetAt = fiveHourResetAt;
    if (self.statusItem != nil) {
        [self refreshCodexQuotaBarTitle];
        [self refreshCodexQuotaTooltip:nil];
    }
    if (self.fiveHourQuotaLabel != nil) {
        [self.fiveHourQuotaLabel setStringValue:fiveHourText];
    }
    if (self.weeklyQuotaLabel != nil) {
        [self.weeklyQuotaLabel setStringValue:weeklyText];
    }
    if (status != nil && self.quotaStatusLabel != nil) {
        [self.quotaStatusLabel setStringValue:status];
    }
}

// 控制是否在顶部栏显示 5 小时额度。
- (void)toggleFiveHourQuota:(id)sender {
    quotaFiveHourVisible = [(NSButton *)sender state] == NSControlStateValueOn;
    [self updateCodexQuota:quotaFiveHourValue weekly:quotaWeeklyValue fiveHourResetAt:quotaFiveHourResetAt status:nil];
}

// 控制是否在顶部栏显示周额度。
- (void)toggleWeeklyQuota:(id)sender {
    quotaWeeklyVisible = [(NSButton *)sender state] == NSControlStateValueOn;
    [self updateCodexQuota:quotaFiveHourValue weekly:quotaWeeklyValue fiveHourResetAt:quotaFiveHourResetAt status:nil];
}

// 应用新的额度刷新间隔，限制在 5 秒到 1 小时，避免误填导致请求过密或过久不更新。
- (void)applyQuotaRefreshInterval:(id)sender {
    (void)sender;
    NSInteger seconds = [self.quotaRefreshSecondsField integerValue];
    if (seconds < 5) {
        seconds = 5;
    }
    if (seconds > 3600) {
        seconds = 3600;
    }
    quotaRefreshSeconds = (int)seconds;
    [self.quotaRefreshSecondsField setIntegerValue:seconds];
    [self.quotaStatusLabel setStringValue:[NSString stringWithFormat:@"刷新间隔已设置为 %ld 秒", (long)seconds]];
}

// 注册 macOS 原生全局热键；Print Screen 在 macOS 上通常以 F13（keyCode 105）传入。
- (void)installPrintScreenHotKey {
    EventTypeSpec eventType = {kEventClassKeyboard, kEventHotKeyPressed};
    InstallApplicationEventHandler(&handlePrintScreenHotKey,
                                   1,
                                   &eventType,
                                   NULL,
                                   &printScreenEventHandlerRef);

    EventHotKeyID hotKeyID = {'SHBA', 1};
    RegisterEventHotKey(105,
                        0,
                        hotKeyID,
                        GetApplicationEventTarget(),
                        0,
                        &printScreenHotKeyRef);

    // 前台窗口使用本地监听作为补充，兼容部分键盘驱动不触发 Carbon 热键的情况。
    NSEventMask keyDownMask = NSEventMaskKeyDown;
    self.localKeyMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:keyDownMask
                                                                      handler:^NSEvent *(NSEvent *event) {
        if (![event isARepeat] && [event keyCode] == 105) {
            printScreenRequested = YES;
        }
        return event;
    }];
    self.globalKeyMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:keyDownMask
                                                                       handler:^(NSEvent *event) {
        if (![event isARepeat] && [event keyCode] == 105) {
            printScreenRequested = YES;
        }
    }];
}

// 创建多个分页：功能 1、截图、额度和 DeepSeek Harness 彼此独立管理。
- (void)buildFeaturePanel {
    NSView *contentView = [self.window contentView];
    NSTabView *tabView = [[NSTabView alloc] initWithFrame:NSMakeRect(0, 0, 600, 360)];

    NSTabViewItem *featureItem = [[NSTabViewItem alloc] initWithIdentifier:@"feature-1"];
    [featureItem setLabel:@"功能 1"];
    NSView *featureView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 600, 330)];
    [featureItem setView:featureView];
    [tabView addTabViewItem:featureItem];

    NSTabViewItem *screenshotItem = [[NSTabViewItem alloc] initWithIdentifier:@"screenshot"];
    [screenshotItem setLabel:@"截图功能"];
    NSView *screenshotView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 600, 330)];
    [screenshotItem setView:screenshotView];
    [tabView addTabViewItem:screenshotItem];

    NSTabViewItem *quotaItem = [[NSTabViewItem alloc] initWithIdentifier:@"codex-quota"];
    [quotaItem setLabel:@"额度设置"];
    NSView *quotaView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 600, 330)];
    [quotaItem setView:quotaView];
    [tabView addTabViewItem:quotaItem];

    NSTabViewItem *deepSeekItem = [[NSTabViewItem alloc] initWithIdentifier:@"deepseek-harness"];
    [deepSeekItem setLabel:@"DeepSeek"];
    NSView *deepSeekView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 600, 330)];
    [deepSeekItem setView:deepSeekView];
    [tabView addTabViewItem:deepSeekItem];

    NSTabViewItem *timerItem = [[NSTabViewItem alloc] initWithIdentifier:@"timer"];
    [timerItem setLabel:@"微微倒计时"];
    NSView *timerView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 600, 330)];
    [timerItem setView:timerView]; [tabView addTabViewItem:timerItem];
    [contentView addSubview:tabView];

    [self buildFeatureTab:featureView];
    [self buildScreenshotTab:screenshotView];
    [self buildQuotaTab:quotaView];
    [self buildDeepSeekTab:deepSeekView];
    [self buildTimerTab:timerView];
}

// 提供三档分钟数编辑框，配置仅接受整数分钟并持久保存。
- (void)buildTimerTab:(NSView *)contentView {
    NSTextField *title = [NSTextField labelWithString:@"微微倒计时：设置顶部菜单的三个分钟档位"]; [title setFont:[NSFont boldSystemFontOfSize:18]];
    [title setFrame:NSMakeRect(32, 280, 536, 30)]; [contentView addSubview:title];
    NSTextField *fields[3];
    for (int i = 0; i < 3; i++) {
        NSString *labelText = [NSString stringWithFormat:@"档位 %d（分钟）", i + 1];
        NSTextField *label = [NSTextField labelWithString:labelText]; [label setFrame:NSMakeRect(48, 226 - i*64, 150, 26)]; [contentView addSubview:label];
        fields[i] = [[NSTextField alloc] initWithFrame:NSMakeRect(220, 224 - i*64, 110, 30)];
        NSInteger configured = [[NSUserDefaults standardUserDefaults] integerForKey:[NSString stringWithFormat:@"timerMinutes%d", i]];
        if (configured < 1 || configured > 999) configured = timerMinutes[i];
        timerMinutes[i] = (int)configured; [fields[i] setIntegerValue:configured]; [contentView addSubview:fields[i]];
    }
    self.timerConfig10Field = fields[0]; self.timerConfig20Field = fields[1]; self.timerConfig30Field = fields[2];
    NSButton *save = [NSButton buttonWithTitle:@"保存设置" target:self action:@selector(saveTimerConfiguration:)];
    [save setFrame:NSMakeRect(48, 8, 120, 34)]; [save setBezelStyle:NSBezelStyleRounded]; [contentView addSubview:save];
    NSButton *showBall = [NSButton buttonWithTitle:@"重新显示圆球" target:self action:@selector(showTimerBall:)];
    [showBall setFrame:NSMakeRect(188, 8, 150, 34)]; [showBall setBezelStyle:NSBezelStyleRounded]; [contentView addSubview:showBall];
    NSTextField *hint = [NSTextField labelWithString:@"选择菜单栏中的分钟档位开始；圆球右键可隐藏，隐藏时倒计时继续。"];
    [hint setFrame:NSMakeRect(48, 50, 500, 32)]; [hint setMaximumNumberOfLines:2]; [contentView addSubview:hint];
}

// 构造功能 1 页面，集中展示脚本状态、执行详情和操作按钮。
- (void)buildFeatureTab:(NSView *)contentView {

    NSTextField *titleLabel = [NSTextField labelWithString:@"功能 1：同步每日统计到 emlog"];
    [titleLabel setFont:[NSFont boldSystemFontOfSize:18]];
    [titleLabel setFrame:NSMakeRect(32, 292, 536, 28)];
    [contentView addSubview:titleLabel];

    self.featureStatusLabel = [NSTextField labelWithString:@"状态：运行中\n等待下一次整点/半点执行"];
    [self.featureStatusLabel setFrame:NSMakeRect(32, 238, 536, 44)];
    [self.featureStatusLabel setMaximumNumberOfLines:2];
    [contentView addSubview:self.featureStatusLabel];

    NSTextField *detailTitle = [NSTextField labelWithString:@"执行详情"];
    [detailTitle setFont:[NSFont boldSystemFontOfSize:13]];
    [detailTitle setFrame:NSMakeRect(32, 207, 456, 20)];
    [contentView addSubview:detailTitle];

    NSScrollView *detailScrollView = [[NSScrollView alloc] initWithFrame:NSMakeRect(32, 76, 536, 120)];
    [detailScrollView setBorderType:NSBezelBorder];
    [detailScrollView setHasVerticalScroller:YES];
    self.featureDetailView = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, 520, 110)];
    [self.featureDetailView setEditable:NO];
    [self.featureDetailView setSelectable:YES];
    [self.featureDetailView setString:@"尚未执行。\n脚本将在下一个整点或半点运行。"];
    [detailScrollView setDocumentView:self.featureDetailView];
    [contentView addSubview:detailScrollView];

    self.pauseButton = [NSButton buttonWithTitle:@"暂停" target:self action:@selector(pauseFeature:)];
    [self.pauseButton setFrame:NSMakeRect(32, 25, 100, 34)];
    [self.pauseButton setBezelStyle:NSBezelStyleRounded];
    [contentView addSubview:self.pauseButton];

    self.resumeButton = [NSButton buttonWithTitle:@"继续" target:self action:@selector(resumeFeature:)];
    [self.resumeButton setFrame:NSMakeRect(144, 25, 100, 34)];
    [self.resumeButton setBezelStyle:NSBezelStyleRounded];
    [self.resumeButton setEnabled:NO];
    [contentView addSubview:self.resumeButton];

    self.runNowButton = [NSButton buttonWithTitle:@"立即执行" target:self action:@selector(runFeatureNow:)];
    [self.runNowButton setFrame:NSMakeRect(256, 25, 100, 34)];
    [self.runNowButton setBezelStyle:NSBezelStyleRounded];
    [contentView addSubview:self.runNowButton];

    NSButton *helloButton = [NSButton buttonWithTitle:@"你好" target:self action:@selector(sayHello:)];
    [helloButton setFrame:NSMakeRect(368, 25, 100, 34)];
    [helloButton setBezelStyle:NSBezelStyleRounded];
    [helloButton setKeyEquivalent:@"\r"];
    [contentView addSubview:helloButton];
}

// 构造独立的截图页面，三个开关互不依赖，可以按需组合使用。
- (void)buildScreenshotTab:(NSView *)contentView {
    NSTextField *titleLabel = [NSTextField labelWithString:@"截图功能"];
    [titleLabel setFont:[NSFont boldSystemFontOfSize:18]];
    [titleLabel setFrame:NSMakeRect(32, 270, 536, 28)];
    [contentView addSubview:titleLabel];

    NSTextField *introLabel = [NSTextField labelWithString:@"按下 Print Screen 后，按以下开关执行对应动作："];
    [introLabel setFrame:NSMakeRect(32, 238, 536, 22)];
    [contentView addSubview:introLabel];

    self.screenshotSaveToggle = [NSButton buttonWithTitle:@"保存截图到“下载”目录"
                                                    target:self
                                                    action:@selector(toggleScreenshotSave:)];
    [self.screenshotSaveToggle setButtonType:NSButtonTypeSwitch];
    [self.screenshotSaveToggle setState:NSControlStateValueOn];
    [self.screenshotSaveToggle setFrame:NSMakeRect(32, 190, 300, 26)];
    [contentView addSubview:self.screenshotSaveToggle];

    self.screenshotClipboardToggle = [NSButton buttonWithTitle:@"复制截图到剪贴板"
                                                           target:self
                                                           action:@selector(toggleScreenshotClipboard:)];
    [self.screenshotClipboardToggle setButtonType:NSButtonTypeSwitch];
    [self.screenshotClipboardToggle setState:NSControlStateValueOn];
    [self.screenshotClipboardToggle setFrame:NSMakeRect(32, 150, 300, 26)];
    [contentView addSubview:self.screenshotClipboardToggle];

    self.screenshotSoundToggle = [NSButton buttonWithTitle:@"截图成功后播放清脆提示音"
                                                     target:self
                                                     action:@selector(toggleScreenshotSound:)];
    [self.screenshotSoundToggle setButtonType:NSButtonTypeSwitch];
    [self.screenshotSoundToggle setState:NSControlStateValueOn];
    [self.screenshotSoundToggle setFrame:NSMakeRect(32, 110, 320, 26)];
    [contentView addSubview:self.screenshotSoundToggle];

    self.screenshotStatusLabel = [NSTextField labelWithString:@"状态：等待按下 Print Screen"];
    [self.screenshotStatusLabel setFrame:NSMakeRect(32, 55, 536, 40)];
    [self.screenshotStatusLabel setMaximumNumberOfLines:2];
    [contentView addSubview:self.screenshotStatusLabel];
}

// 构造额度设置页，控制顶部栏两个百分比的显示和刷新频率。
- (void)buildQuotaTab:(NSView *)contentView {
    NSTextField *titleLabel = [NSTextField labelWithString:@"Codex 额度设置"];
    [titleLabel setFont:[NSFont boldSystemFontOfSize:18]];
    [titleLabel setFrame:NSMakeRect(32, 270, 536, 28)];
    [contentView addSubview:titleLabel];

    NSTextField *introLabel = [NSTextField labelWithString:@"顶部栏显示的是当前剩余额度，数据来自本机 Codex 服务。"];
    [introLabel setFrame:NSMakeRect(32, 238, 536, 22)];
    [contentView addSubview:introLabel];

    self.fiveHourQuotaLabel = [NSTextField labelWithString:@"5小时 --%"];
    [self.fiveHourQuotaLabel setFrame:NSMakeRect(32, 194, 220, 24)];
    [self.fiveHourQuotaLabel setFont:[NSFont boldSystemFontOfSize:15]];
    [contentView addSubview:self.fiveHourQuotaLabel];

    self.weeklyQuotaLabel = [NSTextField labelWithString:@"周 --%"];
    [self.weeklyQuotaLabel setFrame:NSMakeRect(280, 194, 220, 24)];
    [self.weeklyQuotaLabel setFont:[NSFont boldSystemFontOfSize:15]];
    [contentView addSubview:self.weeklyQuotaLabel];

    self.fiveHourQuotaToggle = [NSButton buttonWithTitle:@"显示 5 小时额度"
                                                  target:self
                                                  action:@selector(toggleFiveHourQuota:)];
    [self.fiveHourQuotaToggle setButtonType:NSButtonTypeSwitch];
    [self.fiveHourQuotaToggle setState:NSControlStateValueOn];
    [self.fiveHourQuotaToggle setFrame:NSMakeRect(32, 150, 220, 26)];
    [contentView addSubview:self.fiveHourQuotaToggle];

    self.weeklyQuotaToggle = [NSButton buttonWithTitle:@"显示周额度"
                                                 target:self
                                                 action:@selector(toggleWeeklyQuota:)];
    [self.weeklyQuotaToggle setButtonType:NSButtonTypeSwitch];
    [self.weeklyQuotaToggle setState:NSControlStateValueOn];
    [self.weeklyQuotaToggle setFrame:NSMakeRect(280, 150, 220, 26)];
    [contentView addSubview:self.weeklyQuotaToggle];

    NSTextField *intervalLabel = [NSTextField labelWithString:@"刷新间隔（秒）："];
    [intervalLabel setFrame:NSMakeRect(32, 102, 120, 24)];
    [contentView addSubview:intervalLabel];

    self.quotaRefreshSecondsField = [[NSTextField alloc] initWithFrame:NSMakeRect(150, 98, 80, 28)];
    [self.quotaRefreshSecondsField setStringValue:@"20"];
    [self.quotaRefreshSecondsField setAlignment:NSTextAlignmentRight];
    [contentView addSubview:self.quotaRefreshSecondsField];

    NSButton *applyButton = [NSButton buttonWithTitle:@"应用" target:self action:@selector(applyQuotaRefreshInterval:)];
    [applyButton setFrame:NSMakeRect(245, 98, 80, 28)];
    [applyButton setBezelStyle:NSBezelStyleRounded];
    [contentView addSubview:applyButton];

    self.quotaStatusLabel = [NSTextField labelWithString:@"状态：等待第一次读取 Codex 额度"];
    [self.quotaStatusLabel setFrame:NSMakeRect(32, 50, 536, 32)];
    [self.quotaStatusLabel setMaximumNumberOfLines:2];
    [contentView addSubview:self.quotaStatusLabel];
}

// 构造 DeepSeek Harness 页面，集中提供启动、打开、关闭和端口修改四个操作。
- (void)buildDeepSeekTab:(NSView *)contentView {
    NSTextField *titleLabel = [NSTextField labelWithString:@"DeepSeek Harness"];
    [titleLabel setFont:[NSFont boldSystemFontOfSize:18]];
    [titleLabel setFrame:NSMakeRect(32, 270, 536, 28)];
    [contentView addSubview:titleLabel];

    NSTextField *introLabel = [NSTextField labelWithString:@"管理本机 DeepSeek Harness 网页服务：dsh web"];
    [introLabel setFrame:NSMakeRect(32, 238, 536, 22)];
    [contentView addSubview:introLabel];

    NSTextField *commandLabel = [NSTextField labelWithString:@"命令：~/.local/bin/dsh web --host 127.0.0.1"];
    [commandLabel setFrame:NSMakeRect(32, 210, 536, 22)];
    [contentView addSubview:commandLabel];

    NSTextField *portLabel = [NSTextField labelWithString:@"端口："];
    [portLabel setFrame:NSMakeRect(32, 170, 52, 24)];
    [contentView addSubview:portLabel];

    self.deepSeekPortField = [[NSTextField alloc] initWithFrame:NSMakeRect(84, 166, 90, 28)];
    [self.deepSeekPortField setStringValue:@"17891"];
    [self.deepSeekPortField setAlignment:NSTextAlignmentRight];
    [contentView addSubview:self.deepSeekPortField];

    NSButton *changePortButton = [NSButton buttonWithTitle:@"修改端口"
                                                     target:self
                                                     action:@selector(changeDeepSeekPort:)];
    [changePortButton setFrame:NSMakeRect(190, 166, 100, 28)];
    [changePortButton setBezelStyle:NSBezelStyleRounded];
    [contentView addSubview:changePortButton];

    NSButton *startButton = [NSButton buttonWithTitle:@"启动"
                                                target:self
                                                action:@selector(startDeepSeekHarness:)];
    [startButton setFrame:NSMakeRect(32, 116, 100, 34)];
    [startButton setBezelStyle:NSBezelStyleRounded];
    [contentView addSubview:startButton];

    NSButton *openButton = [NSButton buttonWithTitle:@"open"
                                               target:self
                                               action:@selector(openDeepSeekHarness:)];
    [openButton setFrame:NSMakeRect(144, 116, 100, 34)];
    [openButton setBezelStyle:NSBezelStyleRounded];
    [contentView addSubview:openButton];

    NSButton *closeButton = [NSButton buttonWithTitle:@"关闭"
                                                target:self
                                                action:@selector(closeDeepSeekHarness:)];
    [closeButton setFrame:NSMakeRect(256, 116, 100, 34)];
    [closeButton setBezelStyle:NSBezelStyleRounded];
    [contentView addSubview:closeButton];

    self.deepSeekStatusLabel = [NSTextField labelWithString:@"状态：尚未启动"];
    [self.deepSeekStatusLabel setFrame:NSMakeRect(32, 55, 536, 48)];
    [self.deepSeekStatusLabel setMaximumNumberOfLines:2];
    [contentView addSubview:self.deepSeekStatusLabel];
}

// 处理“你好”按钮点击，并用原生 macOS 对话框反馈结果。
- (void)sayHello:(id)sender {
    (void)sender;
    NSAlert *alert = [[NSAlert alloc] init];
    [alert setMessageText:@"你好"];
    [alert setInformativeText:@"这是一个用 Go 调用 macOS Cocoa 的最小示例。"];
    [alert addButtonWithTitle:@"好的"];
    [alert runModal];
}

// 创建菜单栏小图标及其菜单，让窗口关闭后仍可重新打开或彻底退出。
- (void)buildStatusItem {
    self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    // 菜单栏使用独立 PNG，避免 macOS 在 18px 尺寸下缓存或解析 icns 失败。
    NSString *iconPath = [[NSBundle mainBundle] pathForResource:@"StatusIcon" ofType:@"png"];
    NSImage *statusImage = [[NSImage alloc] initWithContentsOfFile:iconPath];
    [statusImage setSize:NSMakeSize(22, 22)];

    // 使用自定义状态栏视图固定排列，确保通知数字永远位于图标右侧。
    ClickableStatusView *clickableView = [[ClickableStatusView alloc] initWithFrame:NSMakeRect(0, 0, 122, 22)];
    self.quotaStatusView = clickableView;
    clickableView.statusItem = self.statusItem;
    self.quotaBarLabel = [NSTextField labelWithString:@"-- -- --"];
    // 文字字形的默认基线略偏上，下移 2 像素后与头像视觉中心对齐。
    [self.quotaBarLabel setFrame:NSMakeRect(0, -2, 61, 22)];
    [self.quotaBarLabel setFont:[NSFont systemFontOfSize:11]];
    [self.quotaBarLabel setAlignment:NSTextAlignmentRight];
    [self.quotaStatusView addSubview:self.quotaBarLabel];

    NSImageView *iconView = [[NSImageView alloc] initWithFrame:NSMakeRect(66, 0, 22, 22)];
    [iconView setImage:statusImage];
    [iconView setImageScaling:NSImageScaleProportionallyUpOrDown];
    [self.quotaStatusView addSubview:iconView];

    self.notificationBarLabel = [NSTextField labelWithString:@"0"];
    [self.notificationBarLabel setFrame:NSMakeRect(93, -2, 29, 22)];
    [self.notificationBarLabel setFont:[NSFont systemFontOfSize:11]];
    [self.notificationBarLabel setAlignment:NSTextAlignmentLeft];
    [self.quotaStatusView addSubview:self.notificationBarLabel];
    [self.statusItem setView:self.quotaStatusView];
    [self refreshCodexQuotaTooltip:nil];
    // 独立刷新提示文字，让“还有多少分钟”不会因为额度请求间隔变长而过时。
    self.quotaTooltipTimer = [NSTimer scheduledTimerWithTimeInterval:20.0
                                                                  target:self
                                                                selector:@selector(refreshCodexQuotaTooltip:)
                                                                userInfo:nil
                                                                 repeats:YES];

    NSMenu *statusMenu = [[NSMenu alloc] initWithTitle:@"书禾宝"];
    self.timerMenuItems = [[NSMutableArray alloc] init];
    for (int i = 0; i < 3; i++) {
        NSInteger configured = [[NSUserDefaults standardUserDefaults] integerForKey:[NSString stringWithFormat:@"timerMinutes%d", i]];
        if (configured >= 1 && configured <= 999) timerMinutes[i] = (int)configured;
        NSMenuItem *preset = [[NSMenuItem alloc] initWithTitle:[NSString stringWithFormat:@"%d 分钟", timerMinutes[i]] action:@selector(startTimerPreset:) keyEquivalent:@""];
        [preset setTarget:self]; [preset setTag:i]; [statusMenu addItem:preset]; [self.timerMenuItems addObject:preset];
    }
    self.timerControlSeparator = [NSMenuItem separatorItem];
    [statusMenu addItem:self.timerControlSeparator];
    NSMenuItem *timerStatus = [[NSMenuItem alloc] initWithTitle:@"微微倒计时：未运行" action:nil keyEquivalent:@""];
    [timerStatus setEnabled:NO]; [timerStatus setHidden:YES]; [statusMenu addItem:timerStatus]; [self.timerMenuItems addObject:timerStatus];
    NSMenuItem *timerPause = [[NSMenuItem alloc] initWithTitle:@"暂停" action:@selector(pauseTimer:) keyEquivalent:@""];
    [timerPause setTarget:self]; [timerPause setEnabled:NO]; [timerPause setHidden:YES]; [statusMenu addItem:timerPause]; [self.timerMenuItems addObject:timerPause];
    NSMenuItem *timerResume = [[NSMenuItem alloc] initWithTitle:@"继续" action:@selector(resumeTimer:) keyEquivalent:@""];
    [timerResume setTarget:self]; [timerResume setEnabled:NO]; [timerResume setHidden:YES]; [statusMenu addItem:timerResume]; [self.timerMenuItems addObject:timerResume];
    NSMenuItem *timerStop = [[NSMenuItem alloc] initWithTitle:@"停止倒计时" action:@selector(stopTimer:) keyEquivalent:@""];
    [timerStop setTarget:self]; [timerStop setEnabled:NO]; [timerStop setHidden:YES]; [statusMenu addItem:timerStop]; [self.timerMenuItems addObject:timerStop];
    [statusMenu addItem:[NSMenuItem separatorItem]];
    self.todayNotificationItem = [[NSMenuItem alloc]
        initWithTitle:[NSString stringWithFormat:@"今日通知%d", todayNotificationCount]
               action:nil
        keyEquivalent:@""];
    [self.todayNotificationItem setEnabled:NO];
    [statusMenu addItem:self.todayNotificationItem];
    [statusMenu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *openItem = [[NSMenuItem alloc] initWithTitle:@"打开窗口"
                                                       action:@selector(showWindow:)
                                                keyEquivalent:@""];
    [openItem setTarget:self];
    [statusMenu addItem:openItem];
    [statusMenu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"彻底退出"
                                                       action:@selector(quitApp:)
                                                keyEquivalent:@""];
    [quitItem setTarget:self];
    [statusMenu addItem:quitItem];
    [self.statusItem setMenu:statusMenu];
    [self refreshTimerMenu];
}
@end

// 从 Go 后台调度器安全地把状态更新投递到 Cocoa 主线程。
static void updateFeatureStatus(const char *statusText, const char *detailText) {
    NSString *status = statusText == NULL ? @"" : [NSString stringWithUTF8String:statusText];
    NSString *detail = detailText == NULL ? @"" : [NSString stringWithUTF8String:detailText];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (helloController != nil) {
            [(HelloController *)helloController updateFeatureStatus:status detail:detail];
        }
    });
}

// 将 Go 侧统计出的今日通知数量安全地投递到 Cocoa 主线程。
static void updateTodayNotificationCount(int count) {
    todayNotificationCount = count;
    dispatch_async(dispatch_get_main_queue(), ^{
        if (helloController != nil) {
            [(HelloController *)helloController updateTodayNotificationCount:count];
        }
    });
}

// 将 Codex 剩余额度安全地更新到主线程，并同步刷新设置页。
static void updateCodexQuota(int fiveHour, int weekly, long long fiveHourResetAt, const char *statusText) {
    NSString *status = statusText == NULL ? @"" : [NSString stringWithUTF8String:statusText];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (helloController != nil) {
            [(HelloController *)helloController updateCodexQuota:fiveHour weekly:weekly fiveHourResetAt:fiveHourResetAt status:status];
        }
    });
}

// 将 DeepSeek Harness 后台控制结果安全地更新到独立分页。
static void updateDeepSeekStatus(const char *statusText) {
    NSString *status = statusText == NULL ? @"" : [NSString stringWithUTF8String:statusText];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (helloController != nil) {
            [(HelloController *)helloController updateDeepSeekStatus:status];
        }
    });
}

// 读取并消费一次性的 DeepSeek 启动请求。
static BOOL featureDeepSeekConsumeStartRequest(void) {
    BOOL requested = deepSeekStartRequested;
    deepSeekStartRequested = NO;
    return requested;
}

// 读取并消费一次性的 DeepSeek 浏览器打开请求。
static BOOL featureDeepSeekConsumeOpenRequest(void) {
    BOOL requested = deepSeekOpenRequested;
    deepSeekOpenRequested = NO;
    return requested;
}

// 读取并消费一次性的 DeepSeek 关闭请求。
static BOOL featureDeepSeekConsumeCloseRequest(void) {
    BOOL requested = deepSeekCloseRequested;
    deepSeekCloseRequested = NO;
    return requested;
}

// 读取并消费一次性的端口修改请求。
static BOOL featureDeepSeekConsumePortChangeRequest(void) {
    BOOL requested = deepSeekPortChangeRequested;
    deepSeekPortChangeRequested = NO;
    return requested;
}

// 读取 GUI 当前填写的 DeepSeek Harness 端口。
static int featureDeepSeekPort(void) {
    return deepSeekPort;
}

// 读取额度监控当前设置的刷新秒数。
static int featureQuotaRefreshIntervalSeconds(void) {
    return quotaRefreshSeconds;
}

// 将截图处理结果安全地更新到截图分页。
static void updateScreenshotStatus(const char *statusText) {
    NSString *status = statusText == NULL ? @"" : [NSString stringWithUTF8String:statusText];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (helloController != nil) {
            [(HelloController *)helloController updateScreenshotStatus:status];
        }
    });
}

// 读取截图分页中“保存截图”开关的当前值。
static BOOL featureScreenshotSaveEnabled(void) {
    return screenshotSaveEnabled;
}

// 读取截图分页中“复制到剪贴板”开关的当前值。
static BOOL featureScreenshotClipboardEnabled(void) {
    return screenshotClipboardEnabled;
}

// 读取截图分页中“播放声音”开关的当前值。
static BOOL featureScreenshotSoundEnabled(void) {
    return screenshotSoundEnabled;
}

// 只检查屏幕录制授权，不主动弹出系统授权窗口，避免每次按键触发权限循环。
static BOOL featureScreenCaptureAccessGranted(void) {
    return CGPreflightScreenCaptureAccess();
}

// 把已保存的 PNG 写入系统剪贴板；AppKit 操作统一放在主线程执行。
static BOOL featureCopyImageToClipboard(const char *imagePath) {
    if (imagePath == NULL) {
        return NO;
    }

    NSString *path = [NSString stringWithUTF8String:imagePath];
    __block BOOL copied = NO;
    void (^copyBlock)(void) = ^{
        NSData *imageData = [NSData dataWithContentsOfFile:path];
        if (imageData == nil || [imageData length] == 0) {
            return;
        }

        NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
        [pasteboard clearContents];
        [pasteboard declareTypes:@[NSPasteboardTypePNG] owner:nil];
        copied = [pasteboard setData:imageData forType:NSPasteboardTypePNG];
    };

    if ([NSThread isMainThread]) {
        copyBlock();
    } else {
        dispatch_sync(dispatch_get_main_queue(), copyBlock);
    }
    return copied;
}

// 接收系统级 F13/Print Screen 热键，并交给 Go 后台线程执行截图。
static OSStatus handlePrintScreenHotKey(EventHandlerCallRef nextHandler,
                                        EventRef event,
                                        void *userData) {
    (void)nextHandler;
    (void)userData;

    EventHotKeyID hotKeyID;
    GetEventParameter(event,
                      kEventParamDirectObject,
                      typeEventHotKeyID,
                      NULL,
                      sizeof(hotKeyID),
                      NULL,
                      &hotKeyID);
    if (hotKeyID.signature == 'SHBA' && hotKeyID.id == 1) {
        printScreenRequested = YES;
    }
    return noErr;
}

// 供 Go 后台截图监听器消费一次性的 Print Screen 请求。
static BOOL featurePrintScreenConsumeRequest(void) {
    BOOL requested = printScreenRequested;
    printScreenRequested = NO;
    return requested;
}

// 供 Go 调度器读取 GUI 上的暂停状态。
static BOOL featureSchedulerIsPaused(void) {
    return featureSchedulerPaused;
}

// 供 Go 后台调度器消费一次性的“立即执行”请求。
static BOOL featureSchedulerConsumeImmediateRequest(void) {
    BOOL requested = featureRunNowRequested;
    featureRunNowRequested = NO;
    return requested;
}

// 创建窗口、菜单栏图标、按钮并启动 macOS 原生事件循环。
static void runHelloApp(void) {
    @autoreleasepool {
        NSApplication *app = [NSApplication sharedApplication];
        [app setActivationPolicy:NSApplicationActivationPolicyAccessory];

        NSRect frame = NSMakeRect(0, 0, 600, 360);
        NSUInteger style = NSWindowStyleMaskTitled |
                           NSWindowStyleMaskClosable |
                           NSWindowStyleMaskMiniaturizable;
        NSWindow *window = [[NSWindow alloc] initWithContentRect:frame
                                                         styleMask:style
                                                           backing:NSBackingStoreBuffered
                                                             defer:NO];
        [window setTitle:@"书禾宝"];
        [window center];
        [window setReleasedWhenClosed:NO];

        helloController = [[HelloController alloc] init];
        [(HelloController *)helloController setWindow:window];
        [window setDelegate:helloController];
        [(HelloController *)helloController buildStatusItem];
        [(HelloController *)helloController buildFeaturePanel];
        [(HelloController *)helloController installPrintScreenHotKey];
        [window makeKeyAndOrderFront:nil];
        [app activateIgnoringOtherApps:YES];
        [app run];
    }
}
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// 程序入口：把控制权交给 Cocoa，负责窗口和鼠标事件。
func main() {
	// Cocoa 要求窗口创建和事件循环运行在 macOS 主线程，避免 Go 调度导致偶发崩溃。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 先启动后台调度器，再进入 Cocoa 事件循环，确保 GUI 不会被定时任务阻塞。
	startFeatureScheduler()
	// 启动 Print Screen 监听器，截图保存完成后才会播放提示音。
	startPrintScreenCapture()
	// 在创建菜单前先读取一次，避免菜单初次打开时暂时显示默认的 0。
	refreshTodayNotificationCount()
	// 启动 Codex 5 小时和周额度监控，首次读取会等待 Cocoa 页面初始化完成。
	startCodexQuotaMonitor()
	// 启动 DeepSeek Harness 控制器，处理 GUI 的启动、打开、关闭和端口修改请求。
	startDeepSeekHarnessMonitor()
	C.runHelloApp()
}

// 将后台任务状态和执行详情安全地传给 Cocoa GUI。
func publishFeatureStatus(status string, detail string) {
	cStatus := C.CString(status)
	defer C.free(unsafe.Pointer(cStatus))
	cDetail := C.CString(detail)
	defer C.free(unsafe.Pointer(cDetail))
	C.updateFeatureStatus(cStatus, cDetail)
}

// 将今日通知数量交给 Cocoa 菜单栏显示。
func publishTodayNotificationCount(count int) {
	C.updateTodayNotificationCount(C.int(count))
}

// 读取 GUI 中的暂停按钮状态，避免暂停期间执行外部脚本。
func featureSchedulerIsPaused() bool {
	return bool(C.featureSchedulerIsPaused())
}

// 读取并清除一次性的“立即执行”请求。
func featureSchedulerConsumeImmediateRequest() bool {
	return bool(C.featureSchedulerConsumeImmediateRequest())
}

// 读取并清除一次性的 Print Screen 请求。
func featurePrintScreenConsumeRequest() bool {
	return bool(C.featurePrintScreenConsumeRequest())
}

// 读取截图页面中的“保存到下载目录”开关。
func featureScreenshotSaveEnabled() bool {
	return bool(C.featureScreenshotSaveEnabled())
}

// 读取截图页面中的“复制到剪贴板”开关。
func featureScreenshotClipboardEnabled() bool {
	return bool(C.featureScreenshotClipboardEnabled())
}

// 读取截图页面中的“播放提示音”开关。
func featureScreenshotSoundEnabled() bool {
	return bool(C.featureScreenshotSoundEnabled())
}

// 检查当前固定签名的书禾宝是否已经获得屏幕录制授权。
func featureScreenCaptureAccessGranted() bool {
	return bool(C.featureScreenCaptureAccessGranted())
}

// 把 PNG 路径交给 Cocoa，写入 macOS 系统剪贴板。
func copyImageToClipboard(imagePath string) bool {
	cPath := C.CString(imagePath)
	defer C.free(unsafe.Pointer(cPath))
	return bool(C.featureCopyImageToClipboard(cPath))
}

// 将截图功能的处理结果显示到独立分页。
func publishScreenshotStatus(status string) {
	cStatus := C.CString(status)
	defer C.free(unsafe.Pointer(cStatus))
	C.updateScreenshotStatus(cStatus)
}

// 将 Codex 额度交给 Cocoa 状态栏和额度设置分页。
func publishCodexQuota(fiveHour int, weekly int, fiveHourResetAt int64, status string) {
	cStatus := C.CString(status)
	defer C.free(unsafe.Pointer(cStatus))
	C.updateCodexQuota(C.int(fiveHour), C.int(weekly), C.longlong(fiveHourResetAt), cStatus)
}

// 将 DeepSeek Harness 的后台状态显示到独立功能页。
func publishDeepSeekStatus(status string) {
	cStatus := C.CString(status)
	defer C.free(unsafe.Pointer(cStatus))
	C.updateDeepSeekStatus(cStatus)
}

// 读取并消费 DeepSeek Harness 的启动按钮请求。
func featureDeepSeekConsumeStartRequest() bool {
	return bool(C.featureDeepSeekConsumeStartRequest())
}

// 读取并消费 DeepSeek Harness 的浏览器打开请求。
func featureDeepSeekConsumeOpenRequest() bool {
	return bool(C.featureDeepSeekConsumeOpenRequest())
}

// 读取并消费 DeepSeek Harness 的关闭按钮请求。
func featureDeepSeekConsumeCloseRequest() bool {
	return bool(C.featureDeepSeekConsumeCloseRequest())
}

// 读取并消费端口修改请求。
func featureDeepSeekConsumePortChangeRequest() bool {
	return bool(C.featureDeepSeekConsumePortChangeRequest())
}

// 读取 GUI 当前填写的 DeepSeek Harness 端口。
func featureDeepSeekPort() int {
	return int(C.featureDeepSeekPort())
}

// 读取 GUI 设置的额度刷新间隔。
func featureQuotaRefreshIntervalSeconds() int {
	return int(C.featureQuotaRefreshIntervalSeconds())
}
