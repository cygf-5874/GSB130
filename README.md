# ahocora

流式 Aho-Corasick 多模式匹配库：一份模式集，喂一条字节流，把**全部**命中（含重叠、含跨分块的）都报出来。

- 语言/依赖：Go 1.24，仅标准库，无第三方依赖。
- 构建：`bash scripts/build.sh`（即 `go build ./...`）。
- 自检：`bash scripts/check.sh`（`check/` 是**固定验收程序，勿改**）。
- 既有用例：`go test ./...`。

## 用法

```go
m, err := ahocora.Build([]string{"he", "she", "his", "hers"})
if err != nil { /* 空模式串 */ }

hits := m.Match([]byte("ushers"))
// 流式：
m.Reset()
for _, chunk := range chunks {
    hits = append(hits, m.Feed(chunk)...)
}
```

## 对外契约

1. `Build(patterns []string) (*Matcher, error)` 构建失败指针与转移表；
   在结果与 `patterns` 顺序无关的地方，输出必须**确定**。
2. `Feed(chunk)` 分块喂入与一次性 `Match(all)` **逐字节等价**（含跨分块的命中）：
   把任意分块的 `Feed` 结果按顺序拼接，必须等于对同样字节的 `Match`。
3. **重叠命中全部报告**，按 `(结束位置升序, 模式注册序升序)` 排序。
4. 同一结束位置有多个模式命中时，一个都不能丢。
5. 空模式串非法（`Build` 返回错误）；重复模式注册序取**最先**（后续重复不产生新条目）。
6. **字节安全**：输入是任意字节（可能不是合法 UTF-8），全程按 `byte` 比较，不得按 rune 处理。
7. 流式状态只占 `O(总模式长度)`，**不随输入长度增长**（不得缓存已喂入的整条流）。
8. 纯函数：同一输入多次调用结果相同，无全局可变状态。
9. `patterns` 为空时，任何输入都无命中。

## 目录

```
matcher.go       库实现（当前是空壳：方法体 panic("not implemented")）
matcher_test.go  既有用例
check/main.go    固定验收程序（8 个场景，勿改）
scripts/build.sh 构建
scripts/check.sh 自检入口
```

> 约定：`Match.End` 是命中在整条流中的**结束下标（不含）**，从 0 起；
> `Match.Index` 是模式在 `Build` 参数里的**注册序号**，从 0 起。
