# 小鱼缸（MyGo 独立原型）

用 [MyGo](https://github.com/egoist/mygo) 原生 UI 做的透明桌面小鱼：饱和中深蓝等宽字符鱼（白描边）、上升气泡，点击窗口落下带橙边的金黄 `*` 食物。通常可见 3 条分散的鱼，会追过去吃掉食物，并显示约 1.3 秒的 `chomp` / `gulp`。

这是**软件原型，不是完整成品**，也**不是**对参考视频的逐帧复刻（实现时未观看该视频）。请只把下面「已验证」的条目当成实测结果。

## 运行（Linux 验收二进制）

```sh
/workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-amd64
```

或在项目目录：

```sh
cd /workspace/mygo-desktop-fish
go tool mygo dev
# 或
go run .
```

需要 Go 1.27.3 或更新版本。预期窗口：约 560×380，无边框、**透明背景**、无水缸边框或大控制面板。3 条约 26px 的中深蓝 `><(((*>` / `<*)))><` 分散游动；蓝色 `o` / `O` / `°` 气泡上升。点击空白处投下带橙边的金黄 `*`。左上角有浅色字、较实底色的 **`DRAG`** 条：按住并拖动（不要只单击）。同食追赶时，最近的一条对准食物，其余从侧旁接近，避免叠成一串字符。

## 键盘

窗口有焦点时：

- **Esc**、**Ctrl+Alt+I**（macOS 为 **⌘⌥I**）：**无条件恢复交互**（`SetIgnoreMouseEvents(false)`），不会在第二次按下时重新开启穿透
- **Ctrl+Alt+T**（macOS 为 **⌘⌥T**）：**切换**鼠标穿透（这是唯一的 toggle 快捷键）
- **Ctrl+Q** 或 **Ctrl+Alt+Q**（macOS 为 **⌘Q** / **⌘⌥Q**）：退出

全局快捷键与上面相同（窗口未聚焦或已开启穿透时也希望可用）。Wayland 上全局快捷键走桌面门户，可能要确认、被改键，或因缺少已安装的 `.desktop` 而失败；失败时看终端日志。不要把全局快捷键当成所有 Linux 桌面上已验证可用。

## 拖动（已知问题）

第二次 Linux 实测中，可见的 `:::` /「拖动」条多次拖动**窗口仍无位移**。核对 MyGo v0.10.0 的 Linux 实现后：

- 官方 `DragWindow()` 在无边框窗口上不会立刻移动，而是等到指针再移动约 8 像素后调用 `gtk_window_begin_move_drag`。本机是 X11（`DISPLAY=:10`，`xfwm4`），不会走 Wayland 的 layer-shell 移动路径。
- 应用侧已正确给出可点中的拖动区；问题更像是该框架路径在本窗口（透明、无边框、置顶、跳过任务栏）上没有产生位移，而不是单纯「点不中」。
- 本版**不再宣称官方 `DragWindow()` 已在 Linux 上修好**。手柄改为按住后用官方 `Window.SetPosition` 跟随指针。这条路径在本环境**尚未**用新的真实截图验证；Wayland 上合成器也可能忽略 `SetPosition`。

该区域内点击不会投食。

## 托盘（已知问题）

已安装 MyGo 文档中的 Linux 依赖 `libayatana-appindicator3-1`，但第二次实测**面板上仍无图标、无菜单可测**。调查结果（只读，未改桌面配置，也未加装桌面环境）：

- 本机会话总线没有 `org.kde.StatusNotifierWatcher`。
- `xfce4-panel` 当前插件只有应用程序菜单、几个启动器、窗口列表、分隔符和时钟，**没有** `systray` / StatusNotifier 插件（系统里虽有该插件的 `.desktop`，但未加入此面板）。
- 在这种桌面上，`NewTray` 仍可能成功（库已在），但图标无处显示。先前把「托盘创建成功」当成可以关窗隐藏，会把窗口弄丢（再叠加 `SkipTaskbar`）。因此 **Linux 上关闭窗口会退出应用**；macOS/Windows 在托盘创建成功时仍隐藏到托盘（这两边均未实测）。
- 已改为非 template 的蓝色素描图标，以免黑模版在深色面板上隐形；**这不能替代缺失的面板托盘区**。

请用键盘恢复或退出，不要依赖托盘。

## 测试与编译

```sh
gofmt -w .
go test ./...
go vet ./...
go build -o /workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-amd64 .
```

## 产物路径

源码目录：`/workspace/mygo-desktop-fish`

| 文件 | 说明 |
| --- | --- |
| `/workspace/mygo-fish-artifacts/mygo-desktop-fish-src.zip` | 源码（不含 `.git`、缓存和模块） |
| `/workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-amd64` | Linux 可执行文件（本机 `go build`） |
| `/workspace/mygo-fish-artifacts/mygo-desktop-fish-darwin-arm64.zip` | 内含 `小鱼缸.app`（仅编译） |
| `/workspace/mygo-fish-artifacts/mygo-desktop-fish-darwin-amd64.zip` | 内含 `小鱼缸.app`（仅编译） |
| `/workspace/mygo-fish-artifacts/mygo-desktop-fish-windows-amd64.zip` | 内含 `小鱼缸.exe`（仅编译，未签名） |
| `/workspace/mygo-fish-artifacts/mygo-desktop-fish-windows-arm64.zip` | 内含 `小鱼缸.exe`（仅编译，未签名） |

打包目录（未打 zip 的原始输出）仍在 `/workspace/mygo-fish-artifacts/packages/`。交叉编译跳过了 DMG、公证和 Windows 安装包。

## 验证范围

### Linux 已用真实运行确认

截图：`/workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-feed.webp`（第一轮，约 2026-11-24 15:31，Asia/Shanghai）和 `/workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-feed-chomp.webp`（第二轮，约 15:38）。

- 透明背景
- 点击空白处落食
- 第二轮可见 3 条鱼、气泡，以及吞食后的 `chomp`

### Linux 未验证或未解决

- 拖动窗口：第二轮多次拖动手柄，**窗口没有移动**。本版改为 `SetPosition` 跟随，**尚未**再做真实截图。
- 托盘图标与菜单：安装托盘库后仍不可见，原因见上文，**未解决**。
- 全局快捷键、置顶、鼠标穿透在面板托盘上的开关：**未**在 UI 上测完。第二轮观察到 Ctrl+Alt+I 像是 toggle（已改成无条件恢复；**尚未**再实测）。
- 同目标追食时的轻量分离：已改模拟逻辑并有单元测试，**尚未**再截图确认。

### macOS / Windows

仅编译通过，**未在对应系统上运行**。透明、置顶、穿透、托盘、拖动均未实测。Windows 未签名，可能被 SmartScreen 拦截。

未修改 `ybbapp/mygo`、ArcMD 或其他已有仓库；本目录是独立项目，当前不推送 GitHub。
