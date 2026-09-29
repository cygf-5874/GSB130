// Package ahocora 提供流式 Aho-Corasick 多模式匹配。
//
// 匹配器按字节工作：输入可以是任意字节串（可能不是合法 UTF-8），
// 全程以 byte 为单位比较；模式集由 Build 一次给定，之后可以按任意
// 分块 Feed，也可以一次性 Match。
package ahocora

import "errors"

// ErrEmptyPattern 表示模式集中出现了空模式串。
var ErrEmptyPattern = errors.New("ahocora: empty pattern")

// Match 是一次命中。
//
// End 是命中在整条流中的结束位置（不含），从 0 起；Index 是模式在
// Build 参数中的注册序号（从 0 起）；Pattern 是命中的模式原文。
type Match struct {
	End     int
	Index   int
	Pattern string
}

// Matcher 是一个可流式喂入的多模式匹配器。
type Matcher struct {
	patterns []string
}

// Build 依据给定的模式集构建匹配器。
//
// 空模式串非法，返回 ErrEmptyPattern；重复模式只保留最早的注册序号。
func Build(patterns []string) (*Matcher, error) {
	panic("not implemented")
}

// Match 在整段输入上求全部命中，等价于按任意分块 Feed 后汇总。
func (m *Matcher) Match(input []byte) []Match {
	panic("not implemented")
}

// Feed 喂入一个分块，返回在本次分块内新结束的命中。
//
// 返回值里 End 仍是相对整条流的偏移；跨块的命中不会重复报告。
func (m *Matcher) Feed(chunk []byte) []Match {
	panic("not implemented")
}

// Reset 清空流式状态，便于复用同一匹配器处理下一条流。
func (m *Matcher) Reset() {
	panic("not implemented")
}
