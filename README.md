# 小鱼缸（MyGo 独立原型）

用 [MyGo](https://github.com/egoist/mygo) 原生 UI 做的透明桌面小鱼：蓝青色等宽字符鱼、上升气泡，点击窗口落下黄色 `*` 食物，小鱼会追过去吃掉，并短暂显示 `chomp` / `gulp`。

这是软件原型，**不是**对参考视频的逐帧复刻（实现时未观看该视频）。

## 运行（在本机）

需要 Go 1.27.3 或更新版本。在项目目录：

```sh
cd /workspace/mygo-desktop-fish
go tool mygo dev
# 或
go run .
```

预期窗口：无边框、透明背景、无水缸边框或大控制面板；一条 `><(((*>` / `<*)))><` 式蓝青色鱼在游动，蓝色 `o` / `O` / `°` 气泡上升。点击空白处投下黄色 `*`。左上角 `:::` 可拖动窗口。

## 托盘

- **显示鱼缸**：显示窗口，并关闭鼠标穿透。
- **开启鼠标穿透** / **恢复交互（关闭穿透）**：穿透后窗口不接收点击，必须用托盘「恢复交互」或「显示鱼缸」才能再操作或投食。
- **开启/关闭置顶**
- **退出**：真正退出应用。关闭窗口只会隐藏到托盘（若托盘创建失败，关闭窗口可能直接退出）。

## 测试与编译

```sh
gofmt -w .
go test ./...
go vet ./...
go build -o /workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-amd64 .
```

Linux 二进制可在本机有图形界面时运行。交叉编译产物若存在，只表示**编译通过**，**尚未**在 macOS 或 Windows 上验证透明窗口、置顶、鼠标穿透或托盘。

已编译产物目录：`/workspace/mygo-fish-artifacts/`。

## 说明

- 动画按真实时间步长（`dt`）推进，过长停顿会被截断，避免一帧飞出。
- 同时最多 8 颗食物。
- 未修改 `ybbapp/mygo`、ArcMD 或其他已有仓库；本目录是独立项目，当前不推送 GitHub。

## 产物与验证范围

源码目录：`/workspace/mygo-desktop-fish`

可直接运行（Linux amd64，本机已 `go build`）：

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

`go tool mygo build`（跳过 DMG、公证和 Windows 安装包）还在 `/workspace/mygo-fish-artifacts/packages/` 生成了：

- `darwin-arm64/小鱼缸.app`、`darwin-amd64/小鱼缸.app`（仅编译通过；本机是 Linux，**未**在 macOS 上运行，透明窗口、置顶、穿透、托盘均未实测）
- `windows-amd64/小鱼缸.exe`、`windows-arm64/小鱼缸.exe`（仅编译通过；**未**在 Windows 上运行，未签名，可能被 SmartScreen 拦截）
- `linux-amd64/xiao-yu-gang` 及 `.deb` / `.rpm` / `.tar.gz`（已编译；本环境**尚未**打开 GUI 做运行截图）

本环境没有 `libayatana-appindicator3`，Linux 托盘可能创建失败；创建失败时关闭窗口可能直接退出。请勿把编译成功当成 macOS/Windows 透明效果已验证。

源码打包：`/workspace/mygo-fish-artifacts/mygo-desktop-fish-src.zip`（不含 `.git`、缓存和依赖模块）。
