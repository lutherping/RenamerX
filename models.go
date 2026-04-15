package main

// FileItem 代表一个待处理的文件实体
type FileItem struct {
	ID           string `json:"id"`            // 唯一标识 (UUID)
	OriginalName string `json:"originalName"`  // 原文件名 (带扩展名)
	Extension    string `json:"extension"`     // 扩展名 (.txt)
	Path         string `json:"path"`          // 所在目录的绝对路径
	IsDir        bool   `json:"isDir"`         // 是否为文件夹
	NewName      string `json:"newName"`       // 预览/执行后的新文件名
	Status       string `json:"status"`        // 状态: "pending", "success", "error"
	ErrorMsg     string `json:"errorMsg"`      // 错误信息（如果有）
	ModTime      int64  `json:"modTime"`       // 修改时间 (Unix 时间戳毫秒)
	Size         int64  `json:"size"`          // 文件大小 (Bytes)
}

// RenameRule 代表一条重命名规则
type RenameRule struct {
	Type   string            `json:"type"`   // 规则类型
	Params map[string]string `json:"params"` // 规则参数
}
