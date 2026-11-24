# 小鱼缸（MyGo 独立原型）

桌面小组件：一块竖向透明玻璃/亚克力板，落在 menthe 浅蓝底座上；板内是高清等宽字符鱼 `><(((o>` / `<o)))><`、`o`/`O` 气泡、底部亮绿 `{~` / `~}` 水草，点击板内落下黄橙 `o`/`*` 食物。小鱼会追食，并显示约 1.3 秒的 `chomp` / `crunch` / `gulp`。板外和板内背景都是透明的，**没有**圆缸、水面或沙石。

外观依据是参考视频的静态帧（工作副本 `/workspace/mygo-fish-reference/user-fishbowl.mp4`；实现时未观看该视频），并按用户要求把实体板轮廓和浅蓝底座一起画进桌面外形。这是软件原型，**不是完整成品**，也不是硬件复刻。

窗口约 420×520。字符用矢量/系统等宽字体绘制，带很细的辉光，不用低分辨率马赛克放大。

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

需要 Go 1.27.3 或更新版本。点击**板内**投食；按住**浅蓝底座**拖动窗口（标签 `DRAG`）。

## 键盘

- **Esc**、**Ctrl+Alt+I**（macOS 为 **⌘⌥I**）：无条件恢复交互（不会再打开穿透）
- **Ctrl+Alt+T**（macOS 为 **⌘⌥T**）：切换鼠标穿透
- **Ctrl+Q** 或 **Ctrl+Alt+Q**：退出

全局快捷键与上面相同。Wayland 上可能要门户确认或因缺少 `.desktop` 而失败。

## 已知问题（不阻挡本版外观）

- **拖动**：第二次 Linux 实测中，官方 `DragWindow()` 未能移动无边框窗口。本版改由底座上的 `Window.SetPosition` 跟随指针，**尚未**用新外形再截图验证；Wayland 上合成器也可能忽略 `SetPosition`。
- **托盘**：已装 `libayatana-appindicator3-1`，但本机 `xfce4-panel` 没有 systray/StatusNotifier 插件，会话总线也没有 `org.kde.StatusNotifierWatcher`，面板上仍可能无图标。Linux 上关闭窗口会退出，以免隐藏后找不到窗口。
- macOS / Windows 仅编译通过（若产物目录里有对应包），未在对应系统实测。

## 测试与编译

```sh
gofmt -w .
go test ./...
go vet ./...
go build -o /workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-amd64 .
```

源码目录：`/workspace/mygo-desktop-fish`  
Linux 二进制：`/workspace/mygo-fish-artifacts/mygo-desktop-fish-linux-amd64`  
源码包：`/workspace/mygo-fish-artifacts/mygo-desktop-fish-src.zip`

未修改 `ybbapp/mygo`、ArcMD 或其他已有仓库；当前不推送 GitHub。
