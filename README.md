# 小鱼缸（MyGo 独立原型）

用 [MyGo](https://github.com/egoist/mygo) 原生 UI 做的透明桌面小鱼：饱和中深蓝等宽字符鱼（白描边）、上升气泡，点击窗口落下带橙边的金黄 `*` 食物。通常可见 3 条分散的鱼，会追过去吃掉食物，并显示约 1.3 秒的 `chomp` / `gulp`。

这是软件原型，**不是**对参考视频的逐帧复刻（实现时未观看该视频）。当前以改善浅色桌面上的可读性为主，**不把上一轮过淡过小的画面当作完成**。

## 运行（在本机）

需要 Go 1.27.3 或更新版本。验收用已编译二进制（优先）：

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

预期窗口：约 560×380，无边框、**透明背景**、无水缸边框或大控制面板。3 条较大的 `><(((*>` / `<*)))><` 中深蓝鱼（约 26px，白描边）分散游动；蓝色 `o` / `O` / `°` 气泡上升（比鱼弱，但应能看见）。点击空白处投下带橙边的金黄 `*`。左上角有可发现的 **`::: 拖动`** 条：按住并拖动（不要只单击）。无边框 Linux 窗口在指针移动约 8 像素后才会开始移动。

## 拖动与投食

拖动区是带明确尺寸的 `Box`，并调用官方 `DragWindow()`（与 MyGo 示例的标题栏写法一致），不再只靠很小的 `:::` 字形像素。该区域内点击不会投食。若只点一下就松开，窗口不会移动。

## 键盘（避免困在穿透里）

窗口有焦点时：

- **Ctrl+Q**（macOS 为 **⌘Q**）：退出
- **Esc**：恢复交互（关闭鼠标穿透）并显示窗口

全局（即使窗口未聚焦或已开启穿透；在托盘不可用时尤其重要）：

- **Ctrl+Alt+Q**（macOS 为 **⌘⌥Q**）：退出
- **Ctrl+Alt+I**（macOS 为 **⌘⌥I**）：恢复交互并显示窗口

说明：在 Wayland 上，全局快捷键走桌面门户，可能会提示确认、被改键，或因缺少已安装的 `.desktop` 条目而失败。失败时请看终端日志，改用窗口内快捷键（需先能聚焦窗口）或托盘。不要把全局快捷键当成所有 Linux 桌面上已验证可用。

## 托盘

Linux 托盘需要系统包 **`libayatana-appindicator3-1`**（MyGo 官方依赖）。未安装时，创建托盘会失败并在终端打印错误。

- **显示鱼缸**：显示窗口，并关闭鼠标穿透。
- **开启鼠标穿透** / **恢复交互（关闭穿透）**：穿透后窗口不接收点击；用托盘「恢复交互」、**Esc** 或 **Ctrl+Alt+I** 恢复。
- **开启/关闭置顶**
- **退出**（**Ctrl+Q**）：真正退出应用。

关闭窗口：若托盘创建成功，只是隐藏到托盘；若托盘创建失败，关闭窗口会退出应用，以免收不到窗口。

## 测试与编译

```sh
gofmt -w .
go test ./...
go vet ./...
go build -o /workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-amd64 .
```

## 产物与验证范围

源码目录：`/workspace/mygo-desktop-fish`

已编译产物：`/workspace/mygo-fish-artifacts/`  
源码打包：`/workspace/mygo-fish-artifacts/mygo-desktop-fish-src.zip`（不含 `.git`、缓存和依赖模块）。

`go tool mygo build`（跳过 DMG、公证和 Windows 安装包）在 `/workspace/mygo-fish-artifacts/packages/`：

- `darwin-arm64/小鱼缸.app`、`darwin-amd64/小鱼缸.app`：仅编译通过；**未**在 macOS 上运行，透明、置顶、穿透、托盘、拖动均未实测。
- `windows-amd64/小鱼缸.exe`、`windows-arm64/小鱼缸.exe`：仅编译通过；**未**在 Windows 上运行，未签名，可能被 SmartScreen 拦截。
- `linux-amd64/xiao-yu-gang` 及 `.deb` / `.rpm` / `.tar.gz`：已编译。

### 已做的 Linux 运行观察（第一轮截图）

路径：`/workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-feed.webp`（2026-11-24 15:31 左右，Asia/Shanghai）。

已确认：透明背景、点击空白处会落下食物。鱼会换位置，但追食和 `chomp`/`gulp` 当时**没有明确拍到**。左上角旧版 `:::` 两次拖动**未能确认窗口移动**（手柄过小过淡，命中面积不够）。画面整体过淡过小，浅色桌面上几乎看不见，且只有一条鱼。本次已针对这些问题改了视觉和拖动区；**改后的画面尚未做第二次真实截图**。

未修改 `ybbapp/mygo`、ArcMD 或其他已有仓库；本目录是独立项目，当前不推送 GitHub。
