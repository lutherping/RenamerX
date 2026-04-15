package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ScriptItem 用户自定义脚本
type ScriptItem struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// getScriptsFilePath 获取脚本配置文件的路径（与 exe 同目录）
func (a *App) getScriptsFilePath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "scripts.json"
	}
	return filepath.Join(filepath.Dir(exePath), "scripts.json")
}

// LoadScripts 启动时加载已保存的脚本列表
func (a *App) LoadScripts() []ScriptItem {
	data, err := os.ReadFile(a.getScriptsFilePath())
	if err != nil {
		return []ScriptItem{}
	}
	var items []ScriptItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []ScriptItem{}
	}
	return items
}

// SaveScripts 保存脚本列表到文件
func (a *App) SaveScripts(items []ScriptItem) bool {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return false
	}
	return os.WriteFile(a.getScriptsFilePath(), data, 0644) == nil
}


// ============================================================
// 文件选择与列表相关
// ============================================================

// SelectWorkingDirectory 选择工作目录，返回目录路径
func (a *App) SelectWorkingDirectory() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工作目录",
	})
	if err != nil || dir == "" {
		return ""
	}
	return dir
}

// ListSubDirs 列出目录下所有子文件夹
func (a *App) ListSubDirs(dir string) []FileItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	return a.processPaths(paths)
}

// ListSubFiles 列出目录下所有子文件
func (a *App) ListSubFiles(dir string) []FileItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	return a.processPaths(paths)
}

// SelectFiles 呼出系统原生文件选择对话框（手动添加散落的文件）
func (a *App) SelectFiles() []FileItem {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择待处理文件",
	})
	if err != nil {
		return nil
	}
	return a.processPaths(files)
}

// SelectTargetDirectory 选择目标目录（用于 to_folder 规则的输出位置）
func (a *App) SelectTargetDirectory() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择目标输出目录",
	})
	if err != nil || dir == "" {
		return ""
	}
	return dir
}

func (a *App) processPaths(paths []string) []FileItem {
	var items []FileItem
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		items = append(items, FileItem{
			ID:           uuid.New().String(),
			OriginalName: info.Name(),
			Extension:    filepath.Ext(info.Name()),
			Path:         filepath.Dir(p),
			IsDir:        info.IsDir(),
			NewName:      info.Name(),
			Status:       "pending",
			ModTime:      info.ModTime().UnixMilli(),
			Size:         info.Size(),
		})
	}
	return items
}

// ============================================================
// 预览引擎
// ============================================================

// PreviewNames 核心计算逻辑（纯内存计算，不改动物理文件）
func (a *App) PreviewNames(files []FileItem, rules []RenameRule) []FileItem {
	updatedFiles := make([]FileItem, len(files))
	copy(updatedFiles, files)

	for i := range updatedFiles {
		file := &updatedFiles[i]
		newName := file.OriginalName

		for _, rule := range rules {
			newName = a.applyRule(newName, rule, i, file.IsDir)
		}
		file.NewName = newName
	}
	return updatedFiles
}

// applyRule 对单个文件名应用一条规则
// 关键修复: isDir 参数确保文件夹名含 "." 时不会被错误截断
func (a *App) applyRule(name string, rule RenameRule, index int, isDir bool) string {
	var ext, base string
	if isDir {
		// 文件夹：不分离扩展名，整体作为 base
		ext = ""
		base = name
	} else {
		ext = filepath.Ext(name)
		base = name[:len(name)-len(ext)]
	}

	switch rule.Type {

	case "replace":
		find := rule.Params["find"]
		replace := rule.Params["replace"]
		return strings.ReplaceAll(base, find, replace) + ext

	case "prefix":
		prefix := rule.Params["prefix"]
		return prefix + base + ext

	case "suffix":
		suffix := rule.Params["suffix"]
		return base + suffix + ext

	case "insert":
		posStr := rule.Params["pos"]
		text := rule.Params["text"]
		pos, err := strconv.Atoi(posStr)
		if err != nil || pos < 1 || text == "" {
			return name
		}
		pos-- // 1-indexed → 0-indexed
		runes := []rune(base)
		if pos > len(runes) {
			pos = len(runes)
		}
		result := string(runes[:pos]) + text + string(runes[pos:])
		return result + ext

	case "numbering":
		startStr := rule.Params["start"]
		digitsStr := rule.Params["digits"]
		position := rule.Params["position"] // "start" or "end"

		start, _ := strconv.Atoi(startStr)
		digits, _ := strconv.Atoi(digitsStr)
		if digits < 1 {
			digits = 1
		}

		num := start + index
		format := fmt.Sprintf("%%0%dd", digits)
		numStr := fmt.Sprintf(format, num)

		if position == "start" {
			return numStr + base + ext
		}
		return base + numStr + ext

	case "regex":
		find := rule.Params["find"]
		replace := rule.Params["replace"]
		if find == "" {
			return name
		}
		re, err := regexp.Compile(find)
		if err != nil {
			return name
		}
		return re.ReplaceAllString(base, replace) + ext

	case "to_folder":
		// 根据文件名生成同名文件夹
		folderName := base // 默认去掉扩展名
		if rule.Params["keepExt"] == "true" {
			folderName = name // 保留扩展名
		}
		return filepath.Join(folderName, name)

	case "script":
		scriptPath := rule.Params["scriptPath"]
		if scriptPath == "" {
			return name
		}
		if _, err := os.Stat(scriptPath); err != nil {
			return name
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "cmd.exe", "/c", scriptPath, name)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		output, err := cmd.Output()
		if err != nil {
			return name
		}
		return strings.TrimSpace(string(output))

	default:
		return name
	}
}

// ============================================================
// 执行引擎
// ============================================================

// ExecuteRename 执行实际的物理重命名/移动/复制操作
func (a *App) ExecuteRename(files []FileItem, rules []RenameRule) []FileItem {
	// 第一步：始终用最新规则重新计算 newName，防止使用过期数据
	updatedFiles := a.PreviewNames(files, rules)

	// 判断是否包含 to_folder 规则以及其模式
	toFolderMode := "" // "", "create_only", "copy", "move"
	toFolderTarget := ""
	for _, rule := range rules {
		if rule.Type == "to_folder" {
			toFolderMode = rule.Params["mode"]
			if toFolderMode == "" {
				toFolderMode = "move" // 默认移动
			}
			toFolderTarget = rule.Params["targetDir"]
			break
		}
	}

	for i := range updatedFiles {
		item := &updatedFiles[i]
		if item.OriginalName == item.NewName {
			item.Status = "success"
			continue
		}

		oldPath := filepath.Join(item.Path, item.OriginalName)
		var newPath string
		if toFolderTarget != "" {
			// 使用用户指定的目标目录
			newPath = filepath.Join(toFolderTarget, item.NewName)
		} else {
			newPath = filepath.Join(item.Path, item.NewName)
		}

		// 确保目标父目录存在
		err := os.MkdirAll(filepath.Dir(newPath), 0755)
		if err != nil {
			item.Status = "error"
			item.ErrorMsg = "无法创建目录: " + err.Error()
			continue
		}

		if toFolderMode == "copy" {
			// 复制模式：保留原文件
			err = copyFile(oldPath, newPath)
		} else {
			// 移动/重命名模式
			err = os.Rename(oldPath, newPath)
		}

		if err != nil {
			item.Status = "error"
			item.ErrorMsg = err.Error()
		} else {
			item.Status = "success"
			item.OriginalName = item.NewName
		}
	}
	return updatedFiles
}

// copyFile 复制文件（用于 to_folder 的 copy 模式）
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
