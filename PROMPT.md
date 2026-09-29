ahocora 是 Go 1.24 的多模式匹配库，仅标准库，构建走 `scripts/build.sh`，自检走 `scripts/check.sh`。README「对外契约」有 9 条；`matcher.go` 的方法体目前全是 `panic("not implemented")`，既有用例全红。

任务：实现匹配器，让 9 条契约、排序确定性、重叠命中和边界行为全部成立，既有用例转绿。固定件还会检查状态复用、同长度候选的顺序和异常输入，不能只让示例语料通过。

验收：
- go test ./... 全绿；
- bash scripts/check.sh 退出码 0，8 个场景全过。

约束：
1. 不改 `check/`；公开类型与方法签名不变。
2. 只用标准库，不依赖 map 迭代顺序；不得把公开语料硬编码进实现。
3. 长模式、空模式、重叠命中和 Unicode/字节边界都要按 README 处理。
