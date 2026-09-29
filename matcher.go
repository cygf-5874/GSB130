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

// Matcher 是一个可流式喂入的多模式匹配器。
type Matcher struct {
	nodes    []node
	patterns []string

	state int
	pos   int
}

type node struct {
	next [256]int
	fail int
	pat  int
}

func newNode() node {
	n := node{fail: 0, pat: -1}
	for b := range n.next {
		n.next[b] = -1
	}
	return n
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

	m := &Matcher{
		nodes:    []node{newNode()},
		patterns: make([]string, len(patterns)),
		state:    0,
		pos:      0,
	}

	seen := make(map[string]bool, len(patterns))
	for i, p := range patterns {
		m.patterns[i] = p
		if seen[p] {
			continue
		}
		seen[p] = true

		cur := 0
		for j := 0; j < len(p); j++ {
			b := p[j]
			next := m.nodes[cur].next[b]
			if next < 0 {
				next = len(m.nodes)
				m.nodes[cur].next[b] = next
				m.nodes = append(m.nodes, newNode())
			}
			cur = next
		}
		if m.nodes[cur].pat < 0 {
			m.nodes[cur].pat = i
		}
	}

	queue := make([]int, 0, len(m.nodes))
	for b := 0; b < 256; b++ {
		child := m.nodes[0].next[b]
		if child >= 0 {
			m.nodes[child].fail = 0
			queue = append(queue, child)
		}
	}
	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		for b := 0; b < 256; b++ {
			child := m.nodes[cur].next[b]
			if child < 0 {
				continue
			}
			f := m.nodes[cur].fail
			for f != 0 && m.nodes[f].next[b] < 0 {
				f = m.nodes[f].fail
			}
			if next := m.nodes[f].next[b]; next >= 0 && next != child {
				f = next
			}
			m.nodes[child].fail = f
			queue = append(queue, child)
		}
	}

	return m, nil
}

// Match 在整段输入上求全部命中，等价于按任意分块 Feed 后汇总。
func (m *Matcher) Match(input []byte) []Match {
	hits, _ := m.scan(input, 0, 0)
	return hits
}

// Feed 喂入一个分块，返回在本次分块内新结束的命中。
//
// 返回值里 End 仍是相对整条流的偏移；跨块的命中不会重复报告。
func (m *Matcher) Feed(chunk []byte) []Match {
	hits, state := m.scan(chunk, m.state, m.pos)
	m.state = state
	m.pos += len(chunk)
	return hits
}

// Reset 清空流式状态，便于复用同一匹配器处理下一条流。
func (m *Matcher) Reset() {
	m.state = 0
	m.pos = 0
}

func (m *Matcher) scan(input []byte, state, pos int) ([]Match, int) {
	var hits []Match
	var ids []int

	for j := 0; j < len(input); j++ {
		b := input[j]
		cur := state
		for cur != 0 && m.nodes[cur].next[b] < 0 {
			cur = m.nodes[cur].fail
		}
		if next := m.nodes[cur].next[b]; next >= 0 {
			cur = next
		}
		state = cur
		end := pos + j + 1

		ids = ids[:0]
		for s := cur; s != 0; s = m.nodes[s].fail {
			if id := m.nodes[s].pat; id >= 0 {
				ids = append(ids, id)
			}
		}
		sort.Ints(ids)
		for _, id := range ids {
			hits = append(hits, Match{
				End:     end,
				Index:   id,
				Pattern: m.patterns[id],
			})
		}
	}

	return hits, state
}
