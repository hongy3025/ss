# SSH Selector MRU 功能设计文档

**日期：** 2026-06-02
**状态：** 已批准，待实现
**作者：** brainstorming 流程产出

## 1. 目标

为 SSH Selector 添加两个核心功能：
1. **选单循环**：执行 SSH 后不退出，自动返回选单，光标停留在刚才选中的条目
2. **MRU 机制**：记录最近使用的主机，下次启动时按使用时间排序

## 2. 范围

### 2.1 In Scope

- 选单循环：SSH 执行后自动返回选单
- 光标定位：记住上次选中的位置
- MRU 数据持久化：存储在 `~/.ssh-selector/mru.json`
- MRU 排序：所有条目按最近使用时间排序
- MRU 清理：启动时自动清理 config 中不存在的 alias
- 退出方式：仅支持 Ctrl+C 退出

### 2.2 Out of Scope

- 按使用频率排序（仅按时间排序）
- MRU 数据的用户配置（路径固定为 `~/.ssh-selector/mru.json`）
- 选单主题/样式自定义

## 3. 数据模型

### 3.1 MRU 数据结构

```go
package mru

import "time"

type MRUEntry struct {
    LastUsed time.Time `json:"lastUsed"`
    Count    int       `json:"count"`
}

type MRU struct {
    Path    string               `json:"-"`
    Entries map[string]MRUEntry  `json:"entries"`
}
```

### 3.2 JSON 文件格式

```json
{
  "entries": {
    "dev": {
      "lastUsed": "2026-06-02T10:30:00Z",
      "count": 5
    },
    "prod": {
      "lastUsed": "2026-06-02T09:15:00Z",
      "count": 3
    }
  }
}
```

## 4. 组件设计

### 4.1 mru 包

**职责：** 管理 MRU 数据的加载、保存、记录和排序。

**接口：**
```go
type Store interface {
    Load() error
    Save() error
    Record(alias string)
    SortEntries(entries []parser.HostEntry) []parser.HostEntry
    Clean(validAliases map[string]bool)
}

func New(path string) Store
```

**实现要点：**
- `Load()`：从 JSON 文件加载，文件不存在则初始化空 MRU
- `Save()`：写入 JSON 文件，确保目录存在
- `Record()`：更新 `LastUsed` 为当前时间，`Count` 加 1
- `SortEntries()`：按 `LastUsed` 降序排序，未在 MRU 中的条目排在最后
- `Clean()`：删除 `validAliases` 中不存在的 alias

### 4.2 selector 包变更

**当前接口：**
```go
type Provider interface {
    Find(entries []parser.HostEntry) (parser.HostEntry, error)
}
```

**新接口：**
```go
type Provider interface {
    Find(entries []parser.HostEntry, initialIndex int) (parser.HostEntry, int, error)
}
```

**变更说明：**
- 新增 `initialIndex` 参数：指定初始光标位置
- 返回值新增 `int`：返回选中的 index（用于下次定位）

**fuzzyfinder 用法：**
```go
idx, err := fuzzyfinder.Find(
    entries,
    func(i int) string { return entries[i].Display() },
    fuzzyfinder.WithCursorPosition(fuzzyfinder.CursorPosition{Y: initialIndex}),
)
```

### 4.3 main 入口变更

**新流程：**
```go
func run(args []string, stdout, stderr *os.File) int {
    configPath, err := parser.DefaultConfigPath()
    if err != nil { ... }

    // 加载 MRU
    home, _ := os.UserHomeDir()
    mruStore := mru.New(filepath.Join(home, ".ssh-selector", "mru.json"))
    mruStore.Load()

    sel := selector.NewFuzzyFinderProvider()
    currentIndex := 0

    for {
        // 每次循环重新读取 config
        entries, err := parser.ParseFile(configPath)
        if err != nil { ... }

        // 清理 MRU 中的无效 alias
        validAliases := make(map[string]bool)
        for _, e := range entries {
            validAliases[e.Alias] = true
        }
        mruStore.Clean(validAliases)

        // 按 MRU 排序
        sortedEntries := mruStore.SortEntries(entries)

        // 显示选单
        entry, newIndex, err := sel.Find(sortedEntries, currentIndex)
        if err != nil {
            if errors.Is(err, selector.ErrAbort) {
                return 0  // Ctrl+C 退出
            }
            ...
        }

        // 记录 MRU
        mruStore.Record(entry.Alias)
        mruStore.Save()

        // 执行 SSH
        conn := connector.New()
        if err := conn.Connect(entry); err != nil { ... }

        // 更新光标位置
        currentIndex = newIndex
    }
}
```

## 5. 文件结构

```
ssh-selector/
├── cmd/ss/main.go
├── internal/
│   ├── parser/
│   ├── selector/
│   ├── connector/
│   └── mru/
│       ├── mru.go
│       └── mru_test.go
├── go.mod
├── go.sum
└── docs/
    └── superpowers/
        └── specs/
            └── 2026-06-02-ssh-selector-mru-design.md
```

## 6. 排序逻辑

### 6.1 排序规则

1. 按 `LastUsed` 降序排序（最近使用的排最前）
2. 未在 MRU 中的条目排在最后（保持原配置顺序）

### 6.2 排序算法

```go
func (m *MRU) SortEntries(entries []parser.HostEntry) []parser.HostEntry {
    // 分离有 MRU 记录和无记录的条目
    var withMRU, withoutMRU []parser.HostEntry
    for _, e := range entries {
        if _, ok := m.Entries[e.Alias]; ok {
            withMRU = append(withMRU, e)
        } else {
            withoutMRU = append(withoutMRU, e)
        }
    }

    // 对有 MRU 记录的条目按 LastUsed 降序排序
    sort.Slice(withMRU, func(i, j int) bool {
        return m.Entries[withMRU[i].Alias].LastUsed.After(m.Entries[withMRU[j].Alias].LastUsed)
    })

    // 合并
    return append(withMRU, withoutMRU...)
}
```

## 7. 错误处理

| 情况 | 行为 |
|---|---|
| MRU 文件不存在 | 初始化空 MRU，不报错 |
| MRU 文件格式错误 | 初始化空 MRU，stderr 输出警告 |
| MRU 文件写入失败 | stderr 输出警告，不中断程序 |
| SSH config 重新读取失败 | 保留上次的 entries，继续显示选单 |

## 8. 测试策略

| 包 | 测试范围 | 方法 |
|---|---|---|
| mru | Load/Save/Record/SortEntries/Clean | 表格驱动单测 |
| selector | Find 带 initialIndex | mock Provider 注入 |
| main | 选单循环逻辑 | 子进程测试 |

## 9. 依赖

- 第三方：`github.com/ktr0731/go-fuzzyfinder` v0.9.0+（已引入）
- 标准库：`os`、`time`、`encoding/json`、`sort`、`path/filepath`

无新增第三方依赖。

## 10. 后续可考虑（暂不实现）

- 按使用频率排序（可选模式）
- MRU 数据导出/导入
- MRU 数据统计（总使用次数、最常用主机等）
