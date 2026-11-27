# Countdowner NSH 迁移备忘

应用入口、设置表单、数据、定时、绘图和桌面交互都通过共享 Rust 代码实现。`nsh.json` 由 NSH CLI 选择原生后端。`cargo run --features native` 的启动器按目标系统选择 NSH 的 Win32、AppKit 或 GTK 4 crate。应用没有自写 Win32、AppKit 或 GTK 调用。

| 能力 | NSH 0.7.2 的行为 |
| --- | --- |
| 桌面层级 | macOS 支持原生桌面图标层级。Windows 将 `DesktopIcon` 映射到 `HWND_BOTTOM`，NSH 公共 API 无法保证按 Win+D 后仍显示在 Explorer 桌面之上。Linux 的层级、工作区和任务栏行为由窗口管理器决定。 |
| 多显示器 | 从旧配置读取窗口大小和显示器相对坐标。旧的显示器标识用 NSH 的显示器标识哈希匹配。之后的窗口位置由 NSH 的 `restore_bounds` 保存。旧实现运行中临时切换显示器并自动返回的行为有待 NSH 提供统一保证。Linux 上窗口坐标可能由合成器忽略。 |
| 托盘 | 使用 NSH 托盘菜单和图标。Linux 能否显示托盘取决于桌面环境。应用图标由 NSH 管理，旧版随 Windows 任务栏主题切换的单色图标及托盘位置标识没有迁移。 |
| 登录启动 | 使用 NSH 的登录项，显示系统报告的状态。旧的启动注册项由旧版管理。NSH CLI 的开发运行模式不注册登录项。 |
| Windows 发布 | 发布工作流为 MSVC 目标静态链接 C 运行库，沿用单个可执行文件的便携 ZIP。 |
| 相同代码切换后端 | Windows、macOS 和 Linux 使用相同组件和持久化 schema。NSH 仍需要对应平台的运行库与构建工具。桌面小组件行为随 NSH 对目标平台的支持变化。 |
| 界面字体 | 设置页、小组件、日历和语言按钮使用同一设计的 Noto Sans CJK 地区子集。子集包括界面文案用到的字形与拉丁字母、数字。用户输入框和自定义标题使用系统默认字体，切换界面语言时字形和字重不变。 |

上游接口和实现见 [桌面 API](https://nsh.dev/modules/nsh-desktop)、[Windows 窗口层级](https://github.com/longbridge/nsh/blob/v0.7.2/crates/nsh-platform-windows/src/shell.rs#L517-L525)和 [NSH Rust 入口](https://nsh.dev/docs/rust)。
