package main

import (
	"path/filepath"
	"testing"
)

func TestApplyRule(t *testing.T) {
	app := &App{}

	t.Run("Replace", func(t *testing.T) {
		rule := RenameRule{Type: "replace", Params: map[string]string{"find": "old", "replace": "new"}}
		res := app.applyRule("old_file.txt", rule, 0, false)
		if res != "new_file.txt" {
			t.Errorf("Expected new_file.txt, got %s", res)
		}
	})

	t.Run("Prefix", func(t *testing.T) {
		rule := RenameRule{Type: "prefix", Params: map[string]string{"prefix": "PRE_"}}
		res := app.applyRule("file.txt", rule, 0, false)
		if res != "PRE_file.txt" {
			t.Errorf("Expected PRE_file.txt, got %s", res)
		}
	})

	t.Run("Suffix", func(t *testing.T) {
		rule := RenameRule{Type: "suffix", Params: map[string]string{"suffix": "_v2"}}
		res := app.applyRule("file.txt", rule, 0, false)
		if res != "file_v2.txt" {
			t.Errorf("Expected file_v2.txt, got %s", res)
		}
	})

	t.Run("Insert", func(t *testing.T) {
		rule := RenameRule{Type: "insert", Params: map[string]string{"pos": "3", "text": "_INS_"}}
		res := app.applyRule("abcdef.txt", rule, 0, false)
		if res != "ab_INS_cdef.txt" {
			t.Errorf("Expected ab_INS_cdef.txt, got %s", res)
		}
	})

	t.Run("Numbering_Start", func(t *testing.T) {
		rule := RenameRule{Type: "numbering", Params: map[string]string{"start": "1", "digits": "3", "position": "start"}}
		res := app.applyRule("file.txt", rule, 5, false) // 1 + 5 = 6 -> "006"
		if res != "006file.txt" {
			t.Errorf("Expected 006file.txt, got %s", res)
		}
	})

	t.Run("Numbering_End", func(t *testing.T) {
		rule := RenameRule{Type: "numbering", Params: map[string]string{"start": "1", "digits": "2", "position": "end"}}
		res := app.applyRule("file.txt", rule, 0, false)
		if res != "file01.txt" {
			t.Errorf("Expected file01.txt, got %s", res)
		}
	})

	t.Run("Regex", func(t *testing.T) {
		rule := RenameRule{Type: "regex", Params: map[string]string{"find": `(\d+)`, "replace": "NUM"}}
		res := app.applyRule("file123.txt", rule, 0, false)
		if res != "fileNUM.txt" {
			t.Errorf("Expected fileNUM.txt, got %s", res)
		}
	})

	t.Run("ToFolder", func(t *testing.T) {
		rule := RenameRule{Type: "to_folder", Params: map[string]string{}}
		res := app.applyRule("photo.jpg", rule, 0, false)
		expected := filepath.Join("photo", "photo.jpg")
		if res != expected {
			t.Errorf("Expected %s, got %s", expected, res)
		}
	})

	// ===== 关键: 文件夹名含 "." 的边界测试 =====
	t.Run("DirWithDot_Replace", func(t *testing.T) {
		rule := RenameRule{Type: "replace", Params: map[string]string{"find": "v2", "replace": "v3"}}
		res := app.applyRule("project.v2.0", rule, 0, true) // isDir=true
		if res != "project.v3.0" {
			t.Errorf("Expected project.v3.0, got %s", res)
		}
	})

	t.Run("DirWithDot_Suffix", func(t *testing.T) {
		rule := RenameRule{Type: "suffix", Params: map[string]string{"suffix": "_backup"}}
		res := app.applyRule("v2.0", rule, 0, true) // isDir=true
		if res != "v2.0_backup" {
			t.Errorf("Expected v2.0_backup, got %s", res)
		}
	})

	t.Run("DirWithDot_Prefix", func(t *testing.T) {
		rule := RenameRule{Type: "prefix", Params: map[string]string{"prefix": "old_"}}
		res := app.applyRule("v2.0", rule, 0, true)
		if res != "old_v2.0" {
			t.Errorf("Expected old_v2.0, got %s", res)
		}
	})

	// 中文文件名
	t.Run("ChineseFileName_Insert", func(t *testing.T) {
		rule := RenameRule{Type: "insert", Params: map[string]string{"pos": "2", "text": "_标记_"}}
		res := app.applyRule("年度报告.docx", rule, 0, false)
		if res != "年_标记_度报告.docx" {
			t.Errorf("Expected 年_标记_度报告.docx, got %s", res)
		}
	})
}
