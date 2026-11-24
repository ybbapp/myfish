# 小鱼缸（MyGo 独立原型）

桌面小组件：一块竖向透明玻璃/亚克力板，落在 menthe 浅蓝底座上；板内是高清等宽字符鱼 `><(((o>` / `<o)))><`、`o`/`O` 气泡、底部亮绿 `{~` / `~}` 水草。在**板顶上方**的细条（提示 `FEED`）点击，食物从该横向位置落入缸顶并下落。小鱼会追食，并显示约 1.3 秒的 `chomp` / `crunch` / `gulp`。鼠标在**缸面上**停留时，最近的鱼会轻轻朝向并靠近指针（保持距离，不跳动）；悬停不会投食，也不会吞掉随后在上方条上的点击。板面本身**没有填充**（可透出壁纸），窗外空白处鼠标透过。**没有**圆缸、水面或沙石。

外观依据是参考视频的静态帧（工作副本 `/workspace/mygo-fish-reference/user-fishbowl.mp4`；实现时未观看该视频）。这是软件原型，**不是完整成品**，也不是硬件复刻。

窗口约 420×520。字符用矢量/系统等宽字体绘制，带很细的辉光，不用低分辨率马赛克放大。

板面曾一度像浅蓝实心牌，原因是 MyGo 的 `ui.RGBA` 透明度参数是 **0–1**，当时误传了 0–255（例如 28），被夹成完全不透明。现已去掉正面填充，只保留细边线、右侧/顶部少量厚度和高光；其它半透明色已改为 0–1。

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

需要 Go 1.27.3 或更新版本。

- 在板**上方**标有 `FEED` 的细条点击投食，食物从点击的水平位置落入缸内。点击缸面或底座**不会**生成食物。
- 指针停在缸面上：鱼轻微朝向/靠近，移出后恢复闲游。
- 按住**浅蓝底座**尝试拖动（标签 `DRAG`）。

## 键盘

- **Esc**、**Ctrl+Alt+I**（macOS 为 **⌘⌥I**）：无条件恢复交互（不会再打开穿透）
- **Ctrl+Alt+T**（macOS 为 **⌘⌥T**）：切换鼠标穿透
- **Ctrl+Q** 或 **Ctrl+Alt+Q**：退出

全局快捷键与上面相同。Wayland 上可能要门户确认或因缺少 `.desktop` 而失败。输入区域只包括投食条、板和底座，窗外空白处点击透过；要点到板后的桌面元素需打开穿透。

## 已知问题（不阻挡本版外观与投食）

- **拖动（Linux）**：在本机多次实测中，官方 `DragWindow()` 和随后的 `Window.SetPosition` 跟随指针**都未能移动**这个透明无边框窗口。这是已知限制，**不宣称已修好**，本轮不再为此打磨。
- **托盘**：已装 `libayatana-appindicator3-1`，但本机 `xfce4-panel` 没有 systray/StatusNotifier 插件，会话总线也没有 `org.kde.StatusNotifierWatcher`，面板上仍可能无图标。Linux 上关闭窗口会退出，以免隐藏后找不到窗口。
- macOS / Windows 仅编译通过（若产物目录里有对应包），未在对应系统实测。透明、投食、悬停、拖动和托盘在这两套系统上都还没有运行证据。

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
