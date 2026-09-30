// Package ahocora 提供流式 Aho-Corasick 多模式匹配。
//
// 匹配器按字节工作：输入可以是任意字节串（可能不是合法 UTF-8），
// 全程以 byte 为单位比较；模式集由 Build 一次给定，之后可以按任意
// 分块 Feed，也可以一次性 Match。
package ahocora

import (
	"errors"
	"sort"
)

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

// node 是自动机（Trie + 失败指针）上的一个状态。
type node struct {
	// children 是 Trie 的稀疏转移表，按 byte 索引。
	children map[byte]int
	// fail 是该状态的最长真后缀所指向的状态（根记为 0）。
	fail int
	// out 指向沿 fail 链遇到的第一个终态；无则为 0（根非终态）。
	out int
	// patternIndex 是在该状态结束的模式的注册序号；-1 表示非终态。
	patternIndex int
}

// Matcher 是一个可流式喂入的多模式匹配器。
type Matcher struct {
	// patterns 保留 Build 入参的副本，注册序即下标。
	patterns []string
	// nodes 以节点 0 为根。
	nodes []*node

	// 以下两项是流式状态，只随当前这条流推进，与已喂入长度无关：
	// state 是自动机当前状态；offset 是已消费的字节数。
	state  int
	offset int
}

// Build 依据给定的模式集构建匹配器。
//
// 空模式串非法，返回 ErrEmptyPattern；重复模式只保留最早的注册序号。
func Build(patterns []string) (*Matcher, error) {
	for _, p := range patterns {
		if len(p) == 0 {
			return nil, ErrEmptyPattern
		}
	}

	stored := make([]string, len(patterns))
	copy(stored, patterns)

	m := &Matcher{
		patterns: stored,
		nodes:    []*node{{children: make(map[byte]int), patternIndex: -1}},
	}

	seen := make(map[string]bool, len(patterns))
	for idx, pattern := range patterns {
		// 重复模式只保留最早的注册序号，后续重复不产生新条目。
		if seen[pattern] {
			continue
		}
		seen[pattern] = true

		cur := 0
		for i := 0; i < len(pattern); i++ {
			c := pattern[i]
			next, ok := m.nodes[cur].children[c]
			if !ok {
				next = len(m.nodes)
				m.nodes[cur].children[c] = next
				m.nodes = append(m.nodes, &node{
					children:     make(map[byte]int),
					patternIndex: -1,
				})
			}
			cur = next
		}
		if m.nodes[cur].patternIndex < 0 {
			m.nodes[cur].patternIndex = idx
		}
	}

	m.buildFailureLinks()
	return m, nil
}

// buildFailureLinks 用广度优先遍历计算每个状态的 fail 与 out。
func (m *Matcher) buildFailureLinks() {
	root := m.nodes[0]
	queue := make([]int, 0, len(m.nodes))

	// 根的直接子节点：fail 指向根。
	for _, child := range root.children {
		m.nodes[child].fail = 0
		m.nodes[child].out = 0
		queue = append(queue, child)
	}

	for head := 0; head < len(queue); head++ {
		u := queue[head]
		nu := m.nodes[u]
		for c, child := range nu.children {
			f := m.gotoState(nu.fail, c)
			m.nodes[child].fail = f
			if m.nodes[f].patternIndex >= 0 {
				m.nodes[child].out = f
			} else {
				m.nodes[child].out = m.nodes[f].out
			}
			queue = append(queue, child)
		}
	}
}

// gotoState 返回从状态 s 消费字节 c 后到达的状态，跟随失败指针。
func (m *Matcher) gotoState(s int, c byte) int {
	for {
		if next, ok := m.nodes[s].children[c]; ok {
			return next
		}
		if s == 0 {
			return 0
		}
		s = m.nodes[s].fail
	}
}

// emit 收集在状态 s 结束的全部命中，按注册序升序追加到 dst。
//
// 每喂入一个字节只调用一次，且字节是按流顺序处理的，因此不同结束
// 位置的调用天然按 End 升序；同一结束位置处再按 Index 升序排序，
// 整体即满足契约要求的 (End 升序, Index 升序)。
func (m *Matcher) emit(dst []Match, s, end int) []Match {
	var indices []int
	for ; s != 0; s = m.nodes[s].out {
		if idx := m.nodes[s].patternIndex; idx >= 0 {
			indices = append(indices, idx)
		}
	}
	sort.Ints(indices)
	for _, idx := range indices {
		dst = append(dst, Match{End: end, Index: idx, Pattern: m.patterns[idx]})
	}
	return dst
}

// scan 从状态 state、流偏移 offset 起逐字节扫描 input，收集全部命中，
// 并返回扫描结束后的自动机状态。
func (m *Matcher) scan(state, offset int, input []byte) ([]Match, int) {
	var hits []Match
	for _, c := range input {
		state = m.gotoState(state, c)
		offset++
		hits = m.emit(hits, state, offset)
	}
	return hits, state
}

// Match 在整段输入上求全部命中，等价于按任意分块 Feed 后汇总。
//
// 它是只读的：不改变匹配器的流式状态与偏移。
func (m *Matcher) Match(input []byte) []Match {
	hits, _ := m.scan(0, 0, input)
	return hits
}

// Feed 喂入一个分块，返回在本次分块内新结束的命中。
//
// 返回值里 End 仍是相对整条流的偏移；跨块的命中不会重复报告。
func (m *Matcher) Feed(chunk []byte) []Match {
	hits, state := m.scan(m.state, m.offset, chunk)
	m.state = state
	m.offset += len(chunk)
	return hits
}

// Reset 清空流式状态，便于复用同一匹配器处理下一条流。
func (m *Matcher) Reset() {
	m.state = 0
	m.offset = 0
}
