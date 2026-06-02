# SSH Selector 设计文档

**日期：** 2026-06-02
**状态：** 已批准，待实现
**作者：** brainstorming 流程产出

## 1. 目标

构建一个 Go 实现的命令行工具 `ss`，提供类似 fzf 的模糊搜索菜单，让用户从 `~/.ssh/config` 中选择主机入口并快速建立 SSH 连接。工具需同时支持 Windows 与 Linux/macOS，并保留对原 PowerShell 版本 `docs/ss.ps1` 的体验升级。

## 2. 范围

### 2.1 In Scope

- 读取并解析 `~/.ssh/config` 中的顶层 `Host` 块
- 用 `go-fuzzyfinder` 渲染模糊搜索 UI
- 单选模式，匹配作用域为「Host 别名 + 关键元信息」
- 选中后调用系统 `ssh`：
  - Linux/macOS：用 `syscall.Exec` 替换当前进程
  - Windows：自动检测终端（Windows Terminal 优先，fallback `cmd.exe`），在新窗口中执行 `ssh`
- 候选列表展示形如 `alias → user@host:port`

### 2.2 Out of Scope

- SSH config 的 `Include` 指令、`Match` 条件块
- 多选模式
- 主机连接历史记录、自动补全
- 自定义 ssh 参数透传
- 通配符 Host 别名解析（仅按字面过滤）
- TUI 主题、配色自定义
- 配置文件 GUI 编辑

## 3. 项目结构

```
ssh-selector/
├── cmd/ss/
│   └── main.go              # 入口，串联三步
├── internal/
│   ├── parser/              # SSH config 解析
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── selector/            # fuzzyfinder 封装
│   │   ├── selector.go
│   │   └── selector_test.go
│   └── connector/           # 跨平台 ssh 调用
│       ├── connector.go
│       ├── connector_windows.go
│       ├── connector_unix.go
│       └── connector_test.go
├── go.mod                   # module github.com/hongy3025/ss
├── go.sum
├── README.md
└── docs/
    ├── ss.ps1               # 旧版保留作参考
    └── superpowers/
        └── specs/
            └── 2026-06-02-ssh-selector-design.md  # 本文件
```

## 4. 数据模型

```go
package parser

type HostEntry struct {
    Alias        string
    HostName     string
    User         string
    Port         string
    IdentityFile string
}

// Display 拼成 "alias → user@host:port" 形式。
// 缺字段时优雅降级：缺 Port 则省 ":port"；缺 User 则省 "user@"；缺 HostName 则用 Alias。
func (h HostEntry) Display() string
```

## 5. 组件设计

### 5.1 parser 包

**职责：** 把 `~/.ssh/config` 文件解析为 `[]HostEntry`。

**支持的语法子集：**
- 顶层 `Host <patterns...>` 起始块
- 同一块内识别 `HostName` / `User` / `Port` / `IdentityFile` 字段
- `#` 开头与空行跳过
- 多个 `Host` 关键字（少见，但合法）按空格拼接

**不支持：**
- `Include` 指令
- `Match` 条件块
- `key=value` 等号写法
- 块内 `Host` 关键字覆盖（解析时累加）

**核心导出函数：**
```go
func DefaultConfigPath() (string, error)   // 返回 ~/.ssh/config
func Parse(path string) ([]HostEntry, error) // 解析入口
```

**过滤规则：**
- 跳过 alias 含 `*` 或 `?` 的 Host（与 ss.ps1 一致）
- 跳过 alias 完全为 `*` 的默认段

**错误处理：**
- 文件不存在：`fmt.Errorf("ssh config not found at %s", path)`
- 单行解析错误：跳过该块，不中断整个解析
- 文件存在但无有效 Host：返回空切片（main 检查 `len == 0`）

### 5.2 selector 包

**职责：** 用 `go-fuzzyfinder` 渲染选择 UI，返回用户选中的那一个 `HostEntry`。

**接口与实现：**
```go
type Provider interface {
    Find(entries []parser.HostEntry) (parser.HostEntry, error)
}

type FuzzyFinderProvider struct{}

func NewFuzzyFinderProvider() *FuzzyFinderProvider
func (p *FuzzyFinderProvider) Find(entries []parser.HostEntry) (parser.HostEntry, error)
```

**Provider 接口的存在意义：**
- main.go 通过接口依赖，未来可注入 mock
- 后续若要替换其他 fuzzy 库，main 流程零修改

**fuzzyfinder 用法：**
```go
idx, err := fuzzyfinder.Find(
    entries,
    func(i int) string { return entries[i].Display() }, // 展示文本
    func(i int) string { return entries[i].Display() }, // 搜索匹配文本
)
```
两个回调都返回完整 Display 字符串，使别名与元信息都参与模糊匹配。

**用户取消（Esc / Ctrl+C）：**
- fuzzyfinder 返回 `fuzzyfinder.ErrAbort`
- selector 透传该错误，main 安静退出（`os.Exit(0)`）

### 5.3 connector 包

**职责：** 根据操作系统，用合适方式调用系统 `ssh`。

**接口：**
```go
type Connector interface {
    Connect(entry parser.HostEntry) error
}

func New() Connector  // 按 GOOS 返回 WindowsConnector / UnixConnector
```

#### 5.3.1 Linux/macOS 实现

```go
//go:build !windows

func (c *UnixConnector) Connect(entry parser.HostEntry) error {
    sshPath, err := exec.LookPath("ssh")
    if err != nil {
        return fmt.Errorf("ssh not found in PATH: %w", err)
    }
    return syscall.Exec(sshPath, []string{"ssh", entry.Alias}, os.Environ())
}
```

要点：
- 用 `syscall.Exec` 而非 `os/exec.Run`：用户在原 shell 退出后仍在 ssh 会话中，无残留进程
- `[]string{"ssh", alias}` 第一项是 argv[0]（约定）
- 透传完整环境变量

#### 5.3.2 Windows 实现

```go
//go:build windows

func (c *WindowsConnector) Connect(entry parser.HostEntry) error {
    if path, err := exec.LookPath("wt.exe"); err == nil {
        return exec.Command(path, "-d", ".", "ssh", entry.Alias).Start()
    }
    return exec.Command(
        "cmd.exe", "/c", "start", "cmd.exe", "/k", "ssh", entry.Alias,
    ).Start()
}
```

要点：
- `wt.exe -d . ssh <alias>`：以当前目录在 Windows Terminal 新标签中执行
- fallback：`start cmd.exe /k ssh <alias>`：新开 cmd 窗口并保持
- 用 `Start()` 而非 `Run()`：父进程立即返回，不阻塞

#### 5.3.3 构建标签

- `connector_unix.go` 顶部：`//go:build !windows`
- `connector_windows.go` 顶部：`//go:build windows`
- 共享逻辑（接口、工厂、错误定义）放 `connector.go`，无 build tag

### 5.4 main 入口

```go
package main

func main() {
    path, err := parser.DefaultConfigPath()
    if err != nil { die(err) }

    entries, err := parser.Parse(path)
    if err != nil { die(err) }
    if len(entries) == 0 {
        fmt.Fprintln(os.Stderr, "no ssh host entries found")
        os.Exit(1)
    }

    sel := selector.NewFuzzyFinderProvider()
    entry, err := sel.Find(entries)
    if err != nil {
        if errors.Is(err, fuzzyfinder.ErrAbort) {
            os.Exit(0)
        }
        die(err)
    }

    conn := connector.New()
    if err := conn.Connect(entry); err != nil { die(err) }
}

func die(err error) {
    fmt.Fprintln(os.Stderr, "ss:", err)
    os.Exit(1)
}
```

## 6. 错误处理矩阵

| 情况 | 行为 | 退出码 |
|---|---|---|
| `~/.ssh/config` 不存在 | stderr: `ss: ssh config not found at <path>` | 1 |
| config 存在但无有效 Host | stderr: `no ssh host entries found` | 1 |
| fuzzyfinder 用户取消 (Esc) | 静默退出 | 0 |
| 找不到 ssh 命令 | stderr: `ss: ssh not found in PATH: ...` | 1 |
| 正常完成 ssh 调用 | 自然结束（Linux exec 后无返回；Windows Start 立即返回 0） | 0 |

## 7. 依赖

- 第三方：`github.com/ktr0731/go-fuzzyfinder` v0.5.0+
- 标准库：`os`、`os/exec`、`os/user`、`syscall`、`path/filepath`、`bufio`、`strings`、`errors`、`fmt`

无其他第三方依赖，避免供应链风险。

## 8. 测试策略

| 包 | 测试范围 | 方法 |
|---|---|---|
| parser | 解析规则全覆盖 | 表格驱动单测 |
| selector | main 流程验证 | mock Provider 注入 |
| connector | 命令拼接正确性 | 单测 `buildCommand` 纯函数 |
| main | 退出码与错误输出 | 子进程测试（构建 binary 后跑） |

parser 单测是最高优先级——它决定一切输入假设，必须最先稳定。

## 9. 安全考量

- **不展开 SSH config 字段到命令行参数**（如 `-l user -p port`），直接 `ssh <alias>`，让 ssh 自己读 config
- 避免 Go 端做参数转义带来的注入风险
- 与"用户手动 ssh 别名"行为完全一致，便于行为可预测

## 10. 发布与构建

- 标准 `go build` 即可
- 跨平台编译：`GOOS=windows go build` / `GOOS=linux go build` / `GOOS=darwin go build`
- 不引入 Makefile（YAGNI）
- 用户安装：`go install github.com/hongy3025/ss@latest`

## 11. README 覆盖

- 一句话简介
- 终端 UI 文本示意（fuzzyfinder 风格）
- 安装命令
- 用法：`ss`
- 跨平台行为说明（Windows 开新窗口 / Linux 替换进程）
- 与原 `ss.ps1` 的差异
- 链接到 LICENSE

## 12. 后续可考虑（暂不实现）

- 多选模式
- SSH config Include 指令支持
- 连接历史记录
- 自定义快捷键绑定
- 候选列表预览（fuzzyfinder preview 窗口）
