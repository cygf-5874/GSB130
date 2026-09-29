// Command check 是 GSB130 ahocora 的固定验收程序。
//
// 用法：
//
//	go run ./check            跑全部场景
//	go run ./check -list      列出场景名
//	go run ./check --only build  只跑某一组
//
// 输出逐场景 `PASS <组>/<名>` 或 `FAIL <组>/<名>  期望=… 实际=…`，
// 结尾 `结果：通过 x/N`；全过 exit 0，否则 exit 1。失败不早退。
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"ahocora"
)

type assertionError struct {
	expected string
	actual   string
}

func (e *assertionError) Error() string { return "assertion failed" }

func renderMatches(ms []ahocora.Match) string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		parts = append(parts, fmt.Sprintf("{%d,%d,%q}", m.End, m.Index, m.Pattern))
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func expectMatches(expected, actual []ahocora.Match) error {
	equal := len(expected) == len(actual)
	if equal {
		for i := range expected {
			if expected[i] != actual[i] {
				equal = false
				break
			}
		}
	}
	if equal {
		return nil
	}
	return &assertionError{renderMatches(expected), renderMatches(actual)}
}

type scenario struct {
	group string
	name  string
	run   func() error
}

func scenarios() []scenario {
	return []scenario{
		{
			group: "build",
			name:  "basic-match",
			run: func() error {
				m, err := ahocora.Build([]string{"he", "she", "his", "hers"})
				if err != nil {
					return err
				}
				want := []ahocora.Match{
					{End: 4, Index: 0, Pattern: "he"},
					{End: 4, Index: 1, Pattern: "she"},
					{End: 6, Index: 3, Pattern: "hers"},
				}
				return expectMatches(want, m.Match([]byte("ushers")))
			},
		},
		{
			group: "build",
			name:  "duplicate-earliest",
			run: func() error {
				m, err := ahocora.Build([]string{"xy", "ab", "xy"})
				if err != nil {
					return err
				}
				want := []ahocora.Match{
					{End: 2, Index: 0, Pattern: "xy"},
					{End: 4, Index: 1, Pattern: "ab"},
				}
				return expectMatches(want, m.Match([]byte("xyab")))
			},
		},
		{
			group: "stream",
			name:  "chunked-equals-once",
			run: func() error {
				m, err := ahocora.Build([]string{"he", "she", "his", "hers"})
				if err != nil {
					return err
				}
				input := []byte("ushers")
				once := m.Match(input)
				want := []ahocora.Match{
					{End: 4, Index: 0, Pattern: "he"},
					{End: 4, Index: 1, Pattern: "she"},
					{End: 6, Index: 3, Pattern: "hers"},
				}
				if err := expectMatches(want, once); err != nil {
					return err
				}
				m.Reset()
				var streamed []ahocora.Match
				for i := range input {
					streamed = append(streamed, m.Feed(input[i:i+1])...)
				}
				return expectMatches(once, streamed)
			},
		},
		{
			group: "stream",
			name:  "cross-chunk-hit",
			run: func() error {
				m, err := ahocora.Build([]string{"hers", "\xff\x00"})
				if err != nil {
					return err
				}
				input := []byte{'a', 'b', 'h', 'e', 'r', 's', 0xff, 0x00}
				once := m.Match(input)
				want := []ahocora.Match{
					{End: 6, Index: 0, Pattern: "hers"},
					{End: 8, Index: 1, Pattern: "\xff\x00"},
				}
				if err := expectMatches(want, once); err != nil {
					return err
				}
				m.Reset()
				var streamed []ahocora.Match
				streamed = append(streamed, m.Feed(input[:4])...)
				streamed = append(streamed, m.Feed(input[4:6])...)
				streamed = append(streamed, m.Feed(input[6:])...)
				return expectMatches(once, streamed)
			},
		},
		{
			group: "stream",
			name:  "reset-and-reuse",
			run: func() error {
				m, err := ahocora.Build([]string{"aa"})
				if err != nil {
					return err
				}
				first := m.Feed([]byte("aa"))
				if err := expectMatches([]ahocora.Match{{End: 2, Index: 0, Pattern: "aa"}}, first); err != nil {
					return err
				}
				m.Reset()
				second := m.Feed([]byte("aa"))
				return expectMatches(first, second)
			},
		},
		{
			group: "overlap",
			name:  "all-reported",
			run: func() error {
				m, err := ahocora.Build([]string{"aa", "aaa"})
				if err != nil {
					return err
				}
				want := []ahocora.Match{
					{End: 2, Index: 0, Pattern: "aa"},
					{End: 3, Index: 0, Pattern: "aa"},
					{End: 3, Index: 1, Pattern: "aaa"},
					{End: 4, Index: 0, Pattern: "aa"},
					{End: 4, Index: 1, Pattern: "aaa"},
				}
				return expectMatches(want, m.Match([]byte("aaaa")))
			},
		},
		{
			group: "overlap",
			name:  "same-end-multi",
			run: func() error {
				m, err := ahocora.Build([]string{"ab", "bab"})
				if err != nil {
					return err
				}
				want := []ahocora.Match{
					{End: 3, Index: 0, Pattern: "ab"},
					{End: 3, Index: 1, Pattern: "bab"},
				}
				return expectMatches(want, m.Match([]byte("bab")))
			},
		},
		{
			group: "edge",
			name:  "empty-and-illegal",
			run: func() error {
				m, err := ahocora.Build(nil)
				if err != nil {
					return err
				}
				if err := expectMatches(nil, m.Match([]byte("anything"))); err != nil {
					return err
				}
				if _, e := ahocora.Build([]string{""}); e == nil {
					return &assertionError{"Build([\"\"]) 返回错误", "无错误"}
				}
				if _, e := ahocora.Build([]string{"a", ""}); e == nil {
					return &assertionError{"Build([\"a\",\"\"]) 返回错误", "无错误"}
				}
				return nil
			},
		},
	}
}

func runScenario(run func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return run()
}

func main() {
	var only string
	list := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-list" || args[i] == "--list":
			list = true
		case args[i] == "--only":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--only 缺少取值")
				os.Exit(2)
			}
			i++
			only = args[i]
		case strings.HasPrefix(args[i], "--only="):
			only = strings.TrimPrefix(args[i], "--only=")
		default:
			fmt.Fprintln(os.Stderr, "无法识别的参数："+args[i])
			os.Exit(2)
		}
	}

	all := scenarios()

	if list {
		for _, s := range all {
			fmt.Println(s.group + "/" + s.name)
		}
		os.Exit(0)
	}

	passed, ran := 0, 0
	for _, s := range all {
		if only != "" && s.group != only {
			continue
		}
		ran++
		label := s.group + "/" + s.name

		err := runScenario(s.run)
		if err == nil {
			passed++
			fmt.Println("PASS " + label)
			continue
		}

		expected, actual := "(未抛断言)", err.Error()
		var ae *assertionError
		if errors.As(err, &ae) {
			expected, actual = ae.expected, ae.actual
		}
		fmt.Printf("FAIL %s  期望=%s 实际=%s\n", label, expected, actual)
	}

	fmt.Printf("结果：通过 %d/%d\n", passed, ran)
	if passed != ran {
		os.Exit(1)
	}
}
