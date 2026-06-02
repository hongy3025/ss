# ss — SSH selector

一个轻量的 Go CLI 工具，提供类似 fzf 的模糊搜索菜单，从 `~/.ssh/config` 选择主机后跨平台建立 SSH 连接。

## 特性

- 解析 `~/.ssh/config` 顶层 Host 块
- 模糊搜索同时匹配 Host 别名与元信息（`user@host:port`）
- 跨平台：Linux/macOS 替换当前进程；Windows 自动检测 Windows Terminal，未装则 fallback 到 `cmd.exe`
- 单文件二进制，零运行时依赖

## 安装

```bash
go install github.com/hongy3025/ss@latest
```

## 用法

```bash
ss
```

启动后输入关键字筛选，回车确认。

候选列表示例：

```
dev   → root@10.0.0.1:22
prod  → deploy@prod.example.com:2222
```

## 平台差异

| 平台 | 行为 |
|---|---|
| Linux / macOS | `syscall.Exec` 替换当前 shell 进程，ssh 接管终端 |
| Windows | 在新窗口中启动 ssh（Windows Terminal 标签 或 cmd.exe 新窗口） |

## License

MIT
