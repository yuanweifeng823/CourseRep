package httpapi

import (
	"strings"
	"testing"
)

// 省略的 JSON 数组会解码为 nil，pgx 会把它写成 NULL；数据库列是 TEXT[] NOT NULL。
func TestNonNilStringsNormalizesOmittedArrays(t *testing.T) {
	if got := nonNilStrings(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil input should become a non-nil empty slice, got %#v", got)
	}
	values := []string{"选课经验", "学习资源"}
	got := nonNilStrings(values)
	if len(got) != 2 || got[0] != values[0] || got[1] != values[1] {
		t.Fatalf("existing values should be preserved, got %#v", got)
	}
}

func TestValidReviewBounds(t *testing.T) {
	valid := reviewInput{Grading: 1, Workload: 5, Assessment: 3, Usefulness: 4, Body: strings.Repeat("好", 2000)}
	if !validReview(valid) {
		t.Fatal("boundary review should be accepted")
	}
	for _, in := range []reviewInput{
		{Grading: 0, Workload: 3, Assessment: 3, Usefulness: 3},
		{Grading: 6, Workload: 3, Assessment: 3, Usefulness: 3},
		{Grading: 3, Workload: 3, Assessment: 3, Usefulness: 3, Body: strings.Repeat("好", 2001)},
	} {
		if validReview(in) {
			t.Fatalf("invalid review accepted: grading=%d body=%d", in.Grading, len([]rune(in.Body)))
		}
	}
}

func TestValidTagsRejectsBlankAndDuplicates(t *testing.T) {
	if !validTags(nil) || !validTags([]string{"选课经验", "学习资源"}) {
		t.Fatal("empty or distinct tags should be accepted")
	}
	for _, tags := range [][]string{{""}, {"  "}, {"选课经验", "选课经验"}} {
		if validTags(tags) {
			t.Fatalf("invalid tags accepted: %#v", tags)
		}
	}
}

// 初始管理员账号来自默认值或 ADMIN_EMAILS；验证码验证时据此决定 role。
func TestAdminEmailsDefaultAndOverride(t *testing.T) {
	t.Setenv("ADMIN_EMAILS", "")
	if !adminEmails()["weifeyuan2-c@my.cityu.edu.hk"] {
		t.Fatal("default admin email should be included")
	}
	t.Setenv("ADMIN_EMAILS", " First@Example.com , second@example.com ,,")
	got := adminEmails()
	if len(got) != 2 || !got["first@example.com"] || !got["second@example.com"] {
		t.Fatalf("ADMIN_EMAILS should be trimmed, lowercased and split on commas, got %#v", got)
	}
}
