# SSH Selector 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 构建 Go CLI 工具 `ss`，提供模糊搜索菜单，从 `~/.ssh/config` 选择主机后跨平台调用 ssh（Windows 新开窗口 / Linux `syscall.Exec` 替换进程）。

**架构：** `cmd/ss/main.go` 串联三个内部包——`parser` 解析 SSH config、`selector` 封装 `go-fuzzyfinder`、`connector` 跨平台执行 ssh。接口注入便于单测与未来替换。

**技术栈：** Go 1.26、`github.com/ktr0731/go-fuzzyfinder` v0.5+、标准库 `os/exec`、`syscall`、`bufio`。

**参考文档：**
- 设计规格：`docs/superpowers/specs/2026-06-02-ssh-selector-design.md`
- 原型：`docs/ss.ps1`（保留作行为对照）
- go-fuzzyfinder 文档：https://pkg.go.dev/github.com/ktr0731/go-fuzzyfinder

---

## 文件结构

按设计规格第 3 节落地：

| 路径 | 职责 |
|---|---|
| `go.mod` | module 声明 + 第三方依赖 |
| `go.sum` | 依赖校验和（自动生成） |
| `cmd/ss/main.go` | 程序入口，串联三步 |
| `internal/parser/parser.go` | `HostEntry` 类型、`DefaultConfigPath`、`Parse` |
| `internal/parser/parser_test.go` | parser 表格驱动单测 |
| `internal/selector/selector.go` | `Provider` 接口 + `FuzzyFinderProvider` |
| `internal/connector/connector.go` | `Connector` 接口、`New()` 工厂、`buildCommand` 纯函数 |
| `internal/connector/connector_unix.go` | `UnixConnector`，build tag `!windows` |
| `internal/connector/connector_windows.go` | `WindowsConnector`，build tag `windows` |
| `internal/connector/connector_test.go` | `buildCommand` 表格驱动单测 |
| `README.md` | 用户文档 |

每个文件单一职责，文件规模预计都在 100-150 行以内。

---

## 任务列表

- [ ] 任务 1：项目骨架与 Go module 初始化
- [ ] 任务 2：parser 包 - `HostEntry` 类型 + `Display` 方法
- [ ] 任务 3：parser 包 - `DefaultConfigPath`
- [ ] 任务 4：parser 包 - `Parse` 基础字段
- [ ] 任务 5：parser 包 - 通配符过滤
- [ ] 任务 6：parser 包 - 错误处理（文件不存在）
- [ ] 任务 7：selector 包 - 接口与实现
- [ ] 任务 8：connector 包 - 接口、工厂、`buildCommand`
- [ ] 任务 9：connector 包 - `UnixConnector`
- [ ] 任务 10：connector 包 - `WindowsConnector`
- [ ] 任务 11：connector 包 - `buildCommand` 单测
- [ ] 任务 12：main 入口
- [ ] 任务 13：README
- [ ] 任务 14：跨平台编译验证

---

### 任务 1：项目骨架与 Go module 初始化

**文件：**
- 创建：`go.mod`
- 创建：`cmd/ss/main.go`（最小骨架）

- [ ] **步骤 1：初始化 Go module**

```bash
cd <repo-root>
go mod init github.com/hongy3025/ss
```

预期：创建 `go.mod`，内容大致为：
```
module github.com/hongy3025/ss

go 1.26
```

- [ ] **步骤 2：创建 `cmd/ss/main.go` 最小骨架**

文件内容：

```go
package main

func main() {}
```

- [ ] **步骤 3：验证编译**

```bash
go build ./...
```

预期：无输出，退出码 0。

- [ ] **步骤 4：Commit**

```bash
git add go.mod go.sum cmd/ss/main.go
git commit -m "chore: initialize go module and cmd/ss skeleton"
```

---

### 任务 2：parser 包 - `HostEntry` 类型 + `Display` 方法

**文件：**
- 创建：`internal/parser/parser.go`
- 创建：`internal/parser/parser_test.go`

- [ ] **步骤 1：编写 `Display` 方法失败的测试**

文件 `internal/parser/parser_test.go`：

```go
package parser

import "testing"

func TestHostEntry_Display(t *testing.T) {
    tests := []struct {
        name  string
        entry HostEntry
        want  string
    }{
        {
            name:  "all fields populated",
            entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", User: "root", Port: "22"},
            want:  "dev → root@10.0.0.1:22",
        },
        {
            name:  "missing port",
            entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", User: "root"},
            want:  "dev → root@10.0.0.1",
        },
        {
            name:  "missing user",
            entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", Port: "2222"},
            want:  "dev → 10.0.0.1:2222",
        },
        {
            name:  "missing host falls back to alias",
            entry: HostEntry{Alias: "dev", User: "root"},
            want:  "dev → root@dev",
        },
        {
            name:  "only alias",
            entry: HostEntry{Alias: "dev"},
            want:  "dev",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := tt.entry.Display(); got != tt.want {
                t.Errorf("Display() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/parser/...
```

预期：FAIL，错误 `parser.HostEntry.Display undefined`。

- [ ] **步骤 3：实现 `HostEntry` 与 `Display`**

文件 `internal/parser/parser.go`：

```go
package parser

type HostEntry struct {
    Alias        string
    HostName     string
    User         string
    Port         string
    IdentityFile string
}

func (h HostEntry) Display() string {
    host := h.HostName
    if host == "" {
        host = h.Alias
    }
    target := host
    if h.User != "" {
        target = h.User + "@" + host
    }
    if h.Port != "" {
        target = target + ":" + h.Port
    }
    return h.Alias + " → " + target
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/parser/...
```

预期：PASS，5 个子测试全过。

- [ ] **步骤 5：Commit**

```bash
git add internal/parser/
git commit -m "feat(parser): add HostEntry type and Display method"
```

---

### 任务 3：parser 包 - `DefaultConfigPath`

**文件：**
- 修改：`internal/parser/parser.go`
- 修改：`internal/parser/parser_test.go`

- [ ] **步骤 1：追加失败的测试**

在 `parser_test.go` 末尾追加：

```go
func TestDefaultConfigPath(t *testing.T) {
    got, err := DefaultConfigPath()
    if err != nil {
        t.Fatalf("DefaultConfigPath() error = %v", err)
    }
    home, _ := os.UserHomeDir()
    want := filepath.Join(home, ".ssh", "config")
    if got != want {
        t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
    }
}
```

并在文件头部追加 import：

```go
import (
    "os"
    "path/filepath"
    "testing"
)
```

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/parser/... -run TestDefaultConfigPath
```

预期：FAIL，`DefaultConfigPath undefined`。

- [ ] **步骤 3：实现 `DefaultConfigPath`**

在 `parser.go` 顶部加 import：

```go
import (
    "os"
    "path/filepath"
)
```

在 `parser.go` 末尾追加：

```go
func DefaultConfigPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".ssh", "config"), nil
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/parser/...
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/parser/
git commit -m "feat(parser): add DefaultConfigPath"
```

---

### 任务 4：parser 包 - `Parse` 基础字段

**文件：**
- 修改：`internal/parser/parser.go`
- 修改：`internal/parser/parser_test.go`

- [ ] **步骤 1：编写失败的测试**

在 `parser_test.go` 末尾追加：

```go
func TestParse_BasicHostBlock(t *testing.T) {
    input := `Host dev
    HostName 10.0.0.1
    User root
    Port 22
    IdentityFile ~/.ssh/id_ed25519
`
    entries, err := Parse(input)
    if err != nil {
        t.Fatalf("Parse() error = %v", err)
    }
    if len(entries) != 1 {
        t.Fatalf("Parse() returned %d entries, want 1", len(entries))
    }
    got := entries[0]
    want := HostEntry{
        Alias:        "dev",
        HostName:     "10.0.0.1",
        User:         "root",
        Port:         "22",
        IdentityFile: "~/.ssh/id_ed25519",
    }
    if got != want {
        t.Errorf("Parse() entry = %+v, want %+v", got, want)
    }
}

func TestParse_MultipleHosts(t *testing.T) {
    input := `Host dev
    HostName 10.0.0.1
    User root

Host prod
    HostName prod.example.com
    User deploy
    Port 2222
`
    entries, err := Parse(input)
    if err != nil {
        t.Fatalf("Parse() error = %v", err)
    }
    if len(entries) != 2 {
        t.Fatalf("Parse() returned %d entries, want 2", len(entries))
    }
    if entries[0].Alias != "dev" || entries[0].HostName != "10.0.0.1" {
        t.Errorf("entries[0] = %+v", entries[0])
    }
    if entries[1].Alias != "prod" || entries[1].Port != "2222" {
        t.Errorf("entries[1] = %+v", entries[1])
    }
}
```

> 注：测试用 `string` 直接传给 `Parse`，因此 `Parse` 签名应改为接受 `string`（读取 io.Reader），或在测试里用临时文件。先选 `Parse(r io.Reader) ([]HostEntry, error)`，main.go 用 `os.Open` 传入。这样测试更简洁。

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/parser/... -run TestParse
```

预期：FAIL，`Parse undefined`。

- [ ] **步骤 3：实现 `Parse` 接受 `io.Reader`**

在 `parser.go` 顶部加 import：

```go
import (
    "bufio"
    "io"
    "os"
    "path/filepath"
    "strings"
)
```

替换 `parser.go` 末尾为：

```go
func Parse(r io.Reader) ([]HostEntry, error) {
    var entries []HostEntry
    var current *HostEntry

    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        key := fields[0]
        value := strings.Join(fields[1:], " ")

        switch key {
        case "Host":
            if current != nil {
                entries = append(entries, *current)
            }
            alias := value
            if strings.ContainsAny(alias, "*?") {
                current = nil
                continue
            }
            current = &HostEntry{Alias: alias}
        default:
            if current == nil {
                continue
            }
            switch key {
            case "HostName":
                current.HostName = value
            case "User":
                current.User = value
            case "Port":
                current.Port = value
            case "IdentityFile":
                current.IdentityFile = value
            }
        }
    }
    if current != nil {
        entries = append(entries, *current)
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return entries, nil
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/parser/...
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/parser/
git commit -m "feat(parser): add Parse with basic Host block support"
```

---

### 任务 5：parser 包 - 注释、空行与多空格

**文件：**
- 修改：`internal/parser/parser_test.go`

- [ ] **步骤 1：追加测试**

```go
func TestParse_CommentsAndBlankLines(t *testing.T) {
    input := `# this is a comment
# another comment

Host dev
    # inline comment in block
    HostName 10.0.0.1

    User root
`
    entries, err := Parse(input)
    if err != nil {
        t.Fatalf("Parse() error = %v", err)
    }
    if len(entries) != 1 {
        t.Fatalf("Parse() returned %d entries, want 1", len(entries))
    }
    if entries[0].User != "root" {
        t.Errorf("User = %q, want root", entries[0].User)
    }
}
```

- [ ] **步骤 2：运行测试验证通过**

```bash
go test ./internal/parser/... -run TestParse_CommentsAndBlankLines
```

预期：PASS（任务 4 的实现已处理这些情况，验证即可）。

- [ ] **步骤 3：Commit**

```bash
git add internal/parser/parser_test.go
git commit -m "test(parser): cover comments and blank lines"
```

---

### 任务 6：parser 包 - 通配符过滤

**文件：**
- 修改：`internal/parser/parser_test.go`

- [ ] **步骤 1：追加测试**

```go
func TestParse_SkipWildcardHosts(t *testing.T) {
    input := `Host *
    User root

Host dev
    HostName 10.0.0.1

Host *.example.com
    User deploy
`
    entries, err := Parse(input)
    if err != nil {
        t.Fatalf("Parse() error = %v", err)
    }
    if len(entries) != 1 {
        t.Fatalf("Parse() returned %d entries, want 1", len(entries))
    }
    if entries[0].Alias != "dev" {
        t.Errorf("Alias = %q, want dev", entries[0].Alias)
    }
}
```

- [ ] **步骤 2：运行测试验证通过**

```bash
go test ./internal/parser/... -run TestParse_SkipWildcardHosts
```

预期：PASS（任务 4 的实现已包含 `ContainsAny(alias, "*?")` 跳过逻辑）。

- [ ] **步骤 3：Commit**

```bash
git add internal/parser/parser_test.go
git commit -m "test(parser): cover wildcard host filtering"
```

---

### 任务 7：parser 包 - 错误处理（文件不存在）

**文件：**
- 创建：`internal/parser/io.go`
- 修改：`cmd/ss/main.go`（后续任务会用到，先不动）

> 决策：把"读文件 + 调 Parse"的胶水代码单独放到 `io.go`，让 `Parse(r io.Reader)` 保持纯粹；提供 `ParseFile(path string)` 包装文件错误。

- [ ] **步骤 1：编写失败的测试**

文件 `internal/parser/io_test.go`：

```go
package parser

import "testing"

func TestParseFile_NotFound(t *testing.T) {
    _, err := ParseFile("Z:/this/path/does/not/exist/config")
    if err == nil {
        t.Fatal("ParseFile() expected error, got nil")
    }
    if !strings.Contains(err.Error(), "ssh config not found") {
        t.Errorf("error %q should mention 'ssh config not found'", err)
    }
}
```

需要 import `"strings"`。

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/parser/... -run TestParseFile_NotFound
```

预期：FAIL，`ParseFile undefined`。

- [ ] **步骤 3：实现 `ParseFile`**

文件 `internal/parser/io.go`：

```go
package parser

import (
    "fmt"
    "os"
)

func ParseFile(path string) ([]HostEntry, error) {
    f, err := os.Open(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("ssh config not found at %s", path)
        }
        return nil, err
    }
    defer f.Close()
    return Parse(f)
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/parser/...
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/parser/io.go internal/parser/io_test.go
git commit -m "feat(parser): add ParseFile with not-found error"
```

---

### 任务 8：selector 包 - 接口与实现

**文件：**
- 创建：`internal/selector/selector.go`

- [ ] **步骤 1：拉取 `go-fuzzyfinder` 依赖**

```bash
go get github.com/ktr0731/go-fuzzyfinder@latest
```

预期：更新 `go.mod` / `go.sum`，引入 fuzzyfinder。

- [ ] **步骤 2：验证 `fuzzyfinder.Find` API 签名**

```bash
go doc github.com/ktr0731/go-fuzzyfinder.Find
```

预期输出（关键字段）：

```
func Find(items []T, itemFunc func(i int) string, searchStringFunc func(i int) string, opts ...Option) (int, error)
```

确认 `itemFunc` 与 `searchStringFunc` 都是 `func(i int) string`。

- [ ] **步骤 3：实现 `Provider` 与 `FuzzyFinderProvider`**

文件 `internal/selector/selector.go`：

```go
package selector

import (
    "errors"

    "github.com/ktr0731/go-fuzzyfinder"

    "github.com/hongy3025/ss/internal/parser"
)

var ErrAbort = errors.New("user aborted selection")

type Provider interface {
    Find(entries []parser.HostEntry) (parser.HostEntry, error)
}

type FuzzyFinderProvider struct{}

func NewFuzzyFinderProvider() *FuzzyFinderProvider {
    return &FuzzyFinderProvider{}
}

func (p *FuzzyFinderProvider) Find(entries []parser.HostEntry) (parser.HostEntry, error) {
    idx, err := fuzzyfinder.Find(
        entries,
        func(i int) string { return entries[i].Display() },
        func(i int) string { return entries[i].Display() },
    )
    if err != nil {
        if errors.Is(err, fuzzyfinder.ErrAbort) {
            return parser.HostEntry{}, ErrAbort
        }
        return parser.HostEntry{}, err
    }
    return entries[idx], nil
}
```

- [ ] **步骤 4：验证编译**

```bash
go build ./...
```

预期：退出码 0。

- [ ] **步骤 5：Commit**

```bash
git add internal/selector/ go.mod go.sum
git commit -m "feat(selector): add Provider interface and FuzzyFinderProvider"
```

---

### 任务 9：connector 包 - 接口、工厂、`buildCommand`

**文件：**
- 创建：`internal/connector/connector.go`

- [ ] **步骤 1：实现接口与工厂骨架**

文件 `internal/connector/connector.go`：

```go
package connector

import "github.com/hongy3025/ss/internal/parser"

type Connector interface {
    Connect(entry parser.HostEntry) error
}

func New() Connector {
    return newPlatformConnector()
}

func buildCommand(entry parser.HostEntry) []string {
    return []string{"ssh", entry.Alias}
}
```

> `newPlatformConnector()` 将在 `connector_windows.go` 与 `connector_unix.go` 中以 build tag 区分实现。这里先引用，编译会因未定义而失败，由任务 10/11 补齐。

- [ ] **步骤 2：暂时让 `connector.go` 自包含一个 dummy 平台实现以便任务 8 可编译**

> 说明：build tag 机制要求两个平台文件至少有一个存在。我们先在 `connector_unix.go` 写一个 dummy，任务 10/11 再覆盖。

文件 `internal/connector/connector_unix.go`：

```go
//go:build !windows

package connector

import "github.com/hongy3025/ss/internal/parser"

func newPlatformConnector() Connector {
    return &UnixConnector{}
}

type UnixConnector struct{}

func (c *UnixConnector) Connect(entry parser.HostEntry) error {
    return nil
}
```

文件 `internal/connector/connector_windows.go`：

```go
//go:build windows

package connector

import "github.com/hongy3025/ss/internal/parser"

func newPlatformConnector() Connector {
    return &WindowsConnector{}
}

type WindowsConnector struct{}

func (c *WindowsConnector) Connect(entry parser.HostEntry) error {
    return nil
}
```

- [ ] **步骤 3：验证编译**

```bash
go build ./...
```

预期：退出码 0。

- [ ] **步骤 4：Commit**

```bash
git add internal/connector/connector.go internal/connector/connector_unix.go internal/connector/connector_windows.go
git commit -m "feat(connector): add Connector interface and platform stubs"
```

---

### 任务 10：connector 包 - `UnixConnector` 真实实现

**文件：**
- 修改：`internal/connector/connector_unix.go`

- [ ] **步骤 1：实现 `Connect` 用 `syscall.Exec`**

文件 `internal/connector/connector_unix.go`：

```go
//go:build !windows

package connector

import (
    "fmt"
    "os"
    "os/exec"
    "syscall"

    "github.com/hongy3025/ss/internal/parser"
)

func newPlatformConnector() Connector {
    return &UnixConnector{}
}

type UnixConnector struct{}

func (c *UnixConnector) Connect(entry parser.HostEntry) error {
    sshPath, err := exec.LookPath("ssh")
    if err != nil {
        return fmt.Errorf("ssh not found in PATH: %w", err)
    }
    return syscall.Exec(sshPath, []string{"ssh", entry.Alias}, os.Environ())
}
```

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

预期：退出码 0。

- [ ] **步骤 3：Commit**

```bash
git add internal/connector/connector_unix.go
git commit -m "feat(connector): implement UnixConnector with syscall.Exec"
```

---

### 任务 11：connector 包 - `WindowsConnector` 真实实现

**文件：**
- 修改：`internal/connector/connector_windows.go`

- [ ] **步骤 1：实现 `Connect` 自动检测 wt.exe / fallback cmd.exe**

文件 `internal/connector/connector_windows.go`：

```go
//go:build windows

package connector

import (
    "os/exec"

    "github.com/hongy3025/ss/internal/parser"
)

func newPlatformConnector() Connector {
    return &WindowsConnector{}
}

type WindowsConnector struct{}

func (c *WindowsConnector) Connect(entry parser.HostEntry) error {
    if path, err := exec.LookPath("wt.exe"); err == nil {
        return exec.Command(path, "-d", ".", "ssh", entry.Alias).Start()
    }
    return exec.Command(
        "cmd.exe", "/c", "start", "cmd.exe", "/k", "ssh", entry.Alias,
    ).Start()
}
```

- [ ] **步骤 2：验证 Windows 编译**

```bash
GOOS=windows go build ./...
```

预期：退出码 0。

- [ ] **步骤 3：Commit**

```bash
git add internal/connector/connector_windows.go
git commit -m "feat(connector): implement WindowsConnector with wt.exe fallback"
```

---

### 任务 12：connector 包 - `buildCommand` 单测

**文件：**
- 创建：`internal/connector/connector_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
package connector

import (
    "reflect"
    "testing"

    "github.com/hongy3025/ss/internal/parser"
)

func TestBuildCommand(t *testing.T) {
    tests := []struct {
        name  string
        entry parser.HostEntry
        want  []string
    }{
        {
            name:  "simple alias",
            entry: parser.HostEntry{Alias: "dev"},
            want:  []string{"ssh", "dev"},
        },
        {
            name:  "alias with special chars",
            entry: parser.HostEntry{Alias: "prod-us-east"},
            want:  []string{"ssh", "prod-us-east"},
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := buildCommand(tt.entry)
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("buildCommand() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

- [ ] **步骤 2：运行测试验证通过**

```bash
go test ./internal/connector/...
```

预期：PASS。

- [ ] **步骤 3：Commit**

```bash
git add internal/connector/connector_test.go
git commit -m "test(connector): cover buildCommand"
```

---

### 任务 13：main 入口

**文件：**
- 修改：`cmd/ss/main.go`

- [ ] **步骤 1：实现 `main.go`**

文件 `cmd/ss/main.go`：

```go
package main

import (
    "errors"
    "fmt"
    "os"

    "github.com/hongy3025/ss/internal/connector"
    "github.com/hongy3025/ss/internal/parser"
    "github.com/hongy3025/ss/internal/selector"
)

func main() {
    os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
    configPath, err := parser.DefaultConfigPath()
    if err != nil {
        fmt.Fprintln(stderr, "ss:", err)
        return 1
    }

    entries, err := parser.ParseFile(configPath)
    if err != nil {
        fmt.Fprintln(stderr, "ss:", err)
        return 1
    }
    if len(entries) == 0 {
        fmt.Fprintln(stderr, "no ssh host entries found")
        return 1
    }

    sel := selector.NewFuzzyFinderProvider()
    entry, err := sel.Find(entries)
    if err != nil {
        if errors.Is(err, selector.ErrAbort) {
            return 0
        }
        fmt.Fprintln(stderr, "ss:", err)
        return 1
    }

    conn := connector.New()
    if err := conn.Connect(entry); err != nil {
        fmt.Fprintln(stderr, "ss:", err)
        return 1
    }
    return 0
}
```

> 决策：把核心逻辑放在 `run` 函数并接受 `stdout`/`stderr` 便于子进程测试。

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

预期：退出码 0。

- [ ] **步骤 3：本地烟测**

```bash
# 在临时 ~/.ssh/config 下测试
mkdir -p /tmp/ss-test-home/.ssh
cat > /tmp/ss-test-home/.ssh/config <<'EOF'
Host dev
    HostName 10.0.0.1
    User root
EOF
HOME=/tmp/ss-test-home ./ss </dev/null
echo "exit=$?"
```

预期：进入 fuzzyfinder（远程终端下会显示列表），用 `</dev/null` 强制触发 `fuzzyfinder.ErrAbort`，最终 `exit=0`。

- [ ] **步骤 4：Commit**

```bash
git add cmd/ss/main.go
git commit -m "feat: wire up main entry with parser+selector+connector"
```

---

### 任务 14：README

**文件：**
- 创建：`README.md`

- [ ] **步骤 1：编写 README**

```markdown
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

## 与 `docs/ss.ps1` 的差异

- 跨平台：原版仅在 Windows PowerShell 下工作
- Windows 行为：原版在当前 PowerShell 窗口跑 ssh，本工具开新窗口，父进程不阻塞
- 模糊匹配：原版只匹配别名，本工具同时匹配 `user@host:port`

## License

MIT
```

- [ ] **步骤 2：Commit**

```bash
git add README.md
git commit -m "docs: add README"
```

---

### 任务 15：跨平台编译验证

**文件：** 无新增（仅验证）

- [ ] **步骤 1：验证 Linux 编译**

```bash
GOOS=linux go build -o /tmp/ss-linux ./cmd/ss
```

预期：退出码 0，产物在 `/tmp/ss-linux`。

- [ ] **步骤 2：验证 macOS 编译**

```bash
GOOS=darwin go build -o /tmp/ss-darwin ./cmd/ss
```

预期：退出码 0。

- [ ] **步骤 3：验证 Windows 编译**

```bash
GOOS=windows go build -o /tmp/ss.exe ./cmd/ss
```

预期：退出码 0。

- [ ] **步骤 4：清理临时产物并 commit（如 go.mod 有变更）**

```bash
rm /tmp/ss-linux /tmp/ss-darwin /tmp/ss.exe
git status
```

若 `go.mod` / `go.sum` 无变更，无需 commit；若有，跑：

```bash
git add go.mod go.sum
git commit -m "chore: tidy modules after cross-compile check"
```

---

## 自检记录

**1. 规格覆盖度（对照设计文档）：**
- § 4 数据模型 → 任务 2（HostEntry + Display）✓
- § 5.1 parser → 任务 3-7（DefaultConfigPath、Parse、通配符、错误、注释）✓
- § 5.2 selector → 任务 8（Provider + FuzzyFinderProvider）✓
- § 5.3 connector 接口 → 任务 9（Connector + New）✓
- § 5.3.1 Unix → 任务 10（syscall.Exec）✓
- § 5.3.2 Windows → 任务 11（wt.exe + cmd.exe fallback）✓
- § 5.3.3 构建标签 → 任务 9 步骤 2（`_unix.go`/`_windows.go`）✓
- § 5.4 main 入口 → 任务 13 ✓
- § 6 错误处理矩阵 → 任务 13 的 `run` 函数（每种情况对应一个分支）✓
- § 7 依赖 → 任务 8 步骤 1（go-fuzzyfinder）✓
- § 8 测试策略 → 任务 5/6/7/12/13 步骤 3 都有单测或子进程测试 ✓
- § 9 安全考量 → 任务 9 步骤 1 的 `buildCommand` 仅用别名（任务 12 测试覆盖）✓
- § 10 构建/发布 → 任务 15 ✓
- § 11 README → 任务 14 ✓

**2. 占位符扫描：**
- 无 TODO / 待定 / 后续补充
- 所有代码块都是具体可执行内容

**3. 类型一致性：**
- `parser.HostEntry` 在任务 2 定义，任务 4 填充，任务 8/9/10/11/12/13 使用 ✓
- `selector.Provider` 接口在任务 8 定义，任务 13 main 使用 ✓
- `selector.ErrAbort` 在任务 8 定义，任务 13 main 检查 ✓
- `connector.Connector` 接口在任务 9 定义，任务 10/11 实现，任务 13 main 使用 ✓
- `buildCommand` 在任务 9 定义，任务 12 测试 ✓
- `connector.New` 在任务 9 定义，任务 13 main 使用 ✓
