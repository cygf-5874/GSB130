package ahocora

import (
	"reflect"
	"testing"
)

func mustBuild(t *testing.T, patterns ...string) *Matcher {
	t.Helper()
	m, err := Build(patterns)
	if err != nil {
		t.Fatalf("Build(%v) 返回错误: %v", patterns, err)
	}
	return m
}

func TestMatchBasic(t *testing.T) {
	m := mustBuild(t, "he", "she", "his", "hers")

	got := m.Match([]byte("ushers"))
	want := []Match{
		{End: 4, Index: 0, Pattern: "he"},
		{End: 4, Index: 1, Pattern: "she"},
		{End: 6, Index: 3, Pattern: "hers"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match(ushers) = %+v, 期望 %+v", got, want)
	}
}

func TestStreamEqualsOnce(t *testing.T) {
	m := mustBuild(t, "he", "she", "his", "hers")
	input := []byte("ushers")

	once := m.Match(input)

	m.Reset()
	var streamed []Match
	for i := 0; i < len(input); i += 2 {
		end := i + 2
		if end > len(input) {
			end = len(input)
		}
		streamed = append(streamed, m.Feed(input[i:end])...)
	}

	if !reflect.DeepEqual(streamed, once) {
		t.Fatalf("分块结果 %+v 与一次性结果 %+v 不等价", streamed, once)
	}
}

func TestOverlapReported(t *testing.T) {
	m := mustBuild(t, "aa", "aaa")

	got := m.Match([]byte("aaaa"))
	want := []Match{
		{End: 2, Index: 0, Pattern: "aa"},
		{End: 3, Index: 0, Pattern: "aa"},
		{End: 3, Index: 1, Pattern: "aaa"},
		{End: 4, Index: 0, Pattern: "aa"},
		{End: 4, Index: 1, Pattern: "aaa"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("重叠命中 = %+v, 期望 %+v", got, want)
	}
}

func TestEmptyPatternsNoMatch(t *testing.T) {
	m := mustBuild(t)

	if got := m.Match([]byte("anything at all")); len(got) != 0 {
		t.Fatalf("空模式集命中 %+v, 期望无命中", got)
	}
}

func TestBuildRejectsEmptyPattern(t *testing.T) {
	if _, err := Build([]string{"a", ""}); err == nil {
		t.Fatalf("Build 含空模式串应返回错误")
	}
}
