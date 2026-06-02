# SSH Selector MRU 功能实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 SSH Selector 添加选单循环和 MRU（最近使用）功能，让用户执行 SSH 后自动返回选单，并按使用时间排序主机列表。

**架构：** 新增 `internal/mru/` 包管理 MRU 数据持久化；修改 `internal/selector/` 接口支持光标定位；修改 `cmd/ss/main.go` 实现选单循环。

**技术栈：** Go 1.26、`github.com/ktr0731/go-fuzzyfinder` v0.9+、标准库 `encoding/json`、`time`、`sort`。

**参考文档：**
- 设计规格：`docs/superpowers/specs/2026-06-02-ssh-selector-mru-design.md`
- go-fuzzyfinder 文档：https://pkg.go.dev/github.com/ktr0731/go-fuzzyfinder

---

## 文件结构

| 路径 | 职责 | 操作 |
|---|---|---|
| `internal/mru/mru.go` | MRU 数据模型、加载、保存、排序 | 创建 |
| `internal/mru/mru_test.go` | MRU 单测 | 创建 |
| `internal/selector/selector.go` | Provider 接口变更 | 修改 |
| `cmd/ss/main.go` | 选单循环逻辑 | 修改 |

---

## 任务列表

- [ ] 任务 1：mru 包 - 数据模型与 Load/Save
- [ ] 任务 2：mru 包 - Record 方法
- [ ] 任务 3：mru 包 - SortEntries 方法
- [ ] 任务 4：mru 包 - Clean 方法
- [ ] 任务 5：selector 包 - 接口变更
- [ ] 任务 6：main 入口 - 选单循环
- [ ] 任务 7：集成测试验证

---

### 任务 1：mru 包 - 数据模型与 Load/Save

**文件：**
- 创建：`internal/mru/mru.go`
- 创建：`internal/mru/mru_test.go`

- [ ] **步骤 1：编写 Load/Save 失败的测试**

文件 `internal/mru/mru_test.go`：

```go
package mru

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)
	if store == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	mru := store.(*MRU)
	if len(mru.Entries) != 0 {
		t.Errorf("Entries length = %d, want 0", len(mru.Entries))
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)

	// 记录一个条目
	store.Record("dev")

	// 保存
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 重新加载
	store2 := New(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	mru := store2.(*MRU)
	entry, ok := mru.Entries["dev"]
	if !ok {
		t.Fatal("Entries[dev] not found")
	}
	if entry.Count != 1 {
		t.Errorf("Count = %d, want 1", entry.Count)
	}
	if time.Since(entry.LastUsed) > time.Minute {
		t.Errorf("LastUsed is too old: %v", entry.LastUsed)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/mru/...
```

预期：FAIL，`mru.New undefined`。

- [ ] **步骤 3：实现 MRU 数据模型与 Load/Save**

文件 `internal/mru/mru.go`：

```go
package mru

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/hongy3025/ss/internal/parser"
)

type MRUEntry struct {
	LastUsed time.Time `json:"lastUsed"`
	Count    int       `json:"count"`
}

type MRU struct {
	Path    string               `json:"-"`
	Entries map[string]MRUEntry  `json:"entries"`
}

type Store interface {
	Load() error
	Save() error
	Record(alias string)
	SortEntries(entries []parser.HostEntry) []parser.HostEntry
	Clean(validAliases map[string]bool)
}

func New(path string) Store {
	return &MRU{
		Path:    path,
		Entries: make(map[string]MRUEntry),
	}
}

func (m *MRU) Load() error {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded MRU
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil
	}
	m.Entries = loaded.Entries
	return nil
}

func (m *MRU) Save() error {
	dir := filepath.Dir(m.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.Path, data, 0644)
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/mru/...
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/mru/
git commit -m "feat(mru): add MRU data model with Load/Save"
```

---

### 任务 2：mru 包 - Record 方法

**文件：**
- 修改：`internal/mru/mru.go`
- 修改：`internal/mru/mru_test.go`

- [ ] **步骤 1：编写 Record 失败的测试**

在 `mru_test.go` 末尾追加：

```go
func TestRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)

	// 第一次记录
	store.Record("dev")
	mru := store.(*MRU)
	entry := mru.Entries["dev"]
	if entry.Count != 1 {
		t.Errorf("Count = %d, want 1", entry.Count)
	}

	// 第二次记录
	store.Record("dev")
	entry = mru.Entries["dev"]
	if entry.Count != 2 {
		t.Errorf("Count = %d, want 2", entry.Count)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/mru/... -run TestRecord
```

预期：FAIL，`store.Record undefined`。

- [ ] **步骤 3：实现 Record 方法**

在 `mru.go` 末尾追加：

```go
func (m *MRU) Record(alias string) {
	entry := m.Entries[alias]
	entry.LastUsed = time.Now()
	entry.Count++
	m.Entries[alias] = entry
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/mru/... -run TestRecord
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/mru/
git commit -m "feat(mru): add Record method"
```

---

### 任务 3：mru 包 - SortEntries 方法

**文件：**
- 修改：`internal/mru/mru.go`
- 修改：`internal/mru/mru_test.go`

- [ ] **步骤 1：编写 SortEntries 失败的测试**

在 `mru_test.go` 末尾追加：

```go
func TestSortEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)

	// 记录使用顺序：prod 先用，dev 后用
	store.Record("prod")
	time.Sleep(10 * time.Millisecond)
	store.Record("dev")

	entries := []parser.HostEntry{
		{Alias: "dev"},
		{Alias: "prod"},
		{Alias: "staging"},
	}

	sorted := store.SortEntries(entries)

	// 最近使用的应该排最前
	if sorted[0].Alias != "dev" {
		t.Errorf("sorted[0].Alias = %q, want dev", sorted[0].Alias)
	}
	if sorted[1].Alias != "prod" {
		t.Errorf("sorted[1].Alias = %q, want prod", sorted[1].Alias)
	}
	if sorted[2].Alias != "staging" {
		t.Errorf("sorted[2].Alias = %q, want staging", sorted[2].Alias)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/mru/... -run TestSortEntries
```

预期：FAIL，`store.SortEntries undefined`。

- [ ] **步骤 3：实现 SortEntries 方法**

在 `mru.go` 末尾追加：

```go
func (m *MRU) SortEntries(entries []parser.HostEntry) []parser.HostEntry {
	var withMRU, withoutMRU []parser.HostEntry
	for _, e := range entries {
		if _, ok := m.Entries[e.Alias]; ok {
			withMRU = append(withMRU, e)
		} else {
			withoutMRU = append(withoutMRU, e)
		}
	}

	sort.Slice(withMRU, func(i, j int) bool {
		return m.Entries[withMRU[i].Alias].LastUsed.After(m.Entries[withMRU[j].Alias].LastUsed)
	})

	return append(withMRU, withoutMRU...)
}
```

并在 `mru.go` 顶部 import 中添加 `"sort"`。

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/mru/... -run TestSortEntries
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/mru/
git commit -m "feat(mru): add SortEntries method"
```

---

### 任务 4：mru 包 - Clean 方法

**文件：**
- 修改：`internal/mru/mru.go`
- 修改：`internal/mru/mru_test.go`

- [ ] **步骤 1：编写 Clean 失败的测试**

在 `mru_test.go` 末尾追加：

```go
func TestClean(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)

	store.Record("dev")
	store.Record("prod")
	store.Record("old-server")

	validAliases := map[string]bool{
		"dev":  true,
		"prod": true,
	}

	store.Clean(validAliases)

	mru := store.(*MRU)
	if _, ok := mru.Entries["dev"]; !ok {
		t.Error("dev should exist")
	}
	if _, ok := mru.Entries["prod"]; !ok {
		t.Error("prod should exist")
	}
	if _, ok := mru.Entries["old-server"]; ok {
		t.Error("old-server should be cleaned")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
go test ./internal/mru/... -run TestClean
```

预期：FAIL，`store.Clean undefined`。

- [ ] **步骤 3：实现 Clean 方法**

在 `mru.go` 末尾追加：

```go
func (m *MRU) Clean(validAliases map[string]bool) {
	for alias := range m.Entries {
		if !validAliases[alias] {
			delete(m.Entries, alias)
		}
	}
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
go test ./internal/mru/... -run TestClean
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/mru/
git commit -m "feat(mru): add Clean method"
```

---

### 任务 5：selector 包 - 接口变更

**文件：**
- 修改：`internal/selector/selector.go`

- [ ] **步骤 1：修改 Provider 接口**

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
	Find(entries []parser.HostEntry, lastSelectedAlias string) (parser.HostEntry, error)
}

type FuzzyFinderProvider struct{}

func NewFuzzyFinderProvider() *FuzzyFinderProvider {
	return &FuzzyFinderProvider{}
}

func (p *FuzzyFinderProvider) Find(entries []parser.HostEntry, lastSelectedAlias string) (parser.HostEntry, error) {
	opts := []fuzzyfinder.Option{}

	if lastSelectedAlias != "" {
		opts = append(opts, fuzzyfinder.WithPreselected(func(i int) bool {
			return entries[i].Alias == lastSelectedAlias
		}))
	}

	idx, err := fuzzyfinder.Find(
		entries,
		func(i int) string { return entries[i].Display() },
		opts...,
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

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

预期：退出码 0（main.go 会报错，但 selector 包本身应编译通过）。

- [ ] **步骤 3：Commit**

```bash
git add internal/selector/
git commit -m "feat(selector): add lastSelectedAlias parameter for cursor positioning"
```

---

### 任务 6：main 入口 - 选单循环

**文件：**
- 修改：`cmd/ss/main.go`

- [ ] **步骤 1：实现选单循环**

文件 `cmd/ss/main.go`：

```go
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hongy3025/ss/internal/connector"
	"github.com/hongy3025/ss/internal/mru"
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

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(stderr, "ss:", err)
		return 1
	}

	mruPath := filepath.Join(home, ".ssh-selector", "mru.json")
	mruStore := mru.New(mruPath)
	if err := mruStore.Load(); err != nil {
		fmt.Fprintln(stderr, "ss: warning: failed to load MRU:", err)
	}

	sel := selector.NewFuzzyFinderProvider()
	lastSelectedAlias := ""

	for {
		entries, err := parser.ParseFile(configPath)
		if err != nil {
			fmt.Fprintln(stderr, "ss:", err)
			return 1
		}
		if len(entries) == 0 {
			fmt.Fprintln(stderr, "no ssh host entries found")
			return 1
		}

		validAliases := make(map[string]bool)
		for _, e := range entries {
			validAliases[e.Alias] = true
		}
		mruStore.Clean(validAliases)

		sortedEntries := mruStore.SortEntries(entries)

		entry, err := sel.Find(sortedEntries, lastSelectedAlias)
		if err != nil {
			if errors.Is(err, selector.ErrAbort) {
				return 0
			}
			fmt.Fprintln(stderr, "ss:", err)
			return 1
		}

		mruStore.Record(entry.Alias)
		if err := mruStore.Save(); err != nil {
			fmt.Fprintln(stderr, "ss: warning: failed to save MRU:", err)
		}

		conn := connector.New()
		if err := conn.Connect(entry); err != nil {
			fmt.Fprintln(stderr, "ss:", err)
			return 1
		}

		lastSelectedAlias = entry.Alias
	}
}
```

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

预期：退出码 0。

- [ ] **步骤 3：Commit**

```bash
git add cmd/ss/main.go
git commit -m "feat: add menu loop with MRU support"
```

---

### 任务 7：集成测试验证

**文件：** 无新增（仅验证）

- [ ] **步骤 1：运行所有测试**

```bash
go test ./... -v
```

预期：所有测试 PASS。

- [ ] **步骤 2：本地烟测**

```bash
# 在临时 ~/.ssh/config 下测试
mkdir -p /tmp/ss-test-home/.ssh
cat > /tmp/ss-test-home/.ssh/config <<'EOF'
Host dev
    HostName 10.0.0.1
    User root

Host prod
    HostName prod.example.com
    User deploy
EOF
HOME=/tmp/ss-test-home ./ss </dev/null
echo "exit=$?"
```

预期：进入 fuzzyfinder，用 `</dev/null` 强制触发 `fuzzyfinder.ErrAbort`，最终 `exit=0`。

- [ ] **步骤 3：验证 MRU 文件创建**

```bash
cat /tmp/ss-test-home/.ssh-selector/mru.json
```

预期：如果用户在步骤 2 中选择了主机，应看到 MRU 数据。

- [ ] **步骤 4：清理临时产物**

```bash
rm -rf /tmp/ss-test-home
```

---

## 自检记录

**1. 规格覆盖度（对照设计文档）：**
- § 3.1 MRU 数据结构 → 任务 1（MRUEntry、MRU 类型）✓
- § 4.1 mru 包接口 → 任务 1-4（Load/Save/Record/SortEntries/Clean）✓
- § 4.2 selector 包变更 → 任务 5（接口变更）✓
- § 4.3 main 入口变更 → 任务 6（选单循环）✓
- § 6 排序逻辑 → 任务 3（SortEntries 实现）✓
- § 7 错误处理 → 任务 1（Load 文件不存在）、任务 6（MRU 警告）✓

**2. 占位符扫描：**
- 无 TODO / 待定 / 后续补充
- 所有代码块都是具体可执行内容

**3. 类型一致性：**
- `mru.Store` 接口在任务 1 定义，任务 6 main 使用 ✓
- `mru.MRU` 结构体在任务 1 定义，任务 2-4 使用 ✓
- `selector.Provider.Find` 签名在任务 5 变更，任务 6 main 调用 ✓
