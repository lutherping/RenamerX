# RenamerX — 批量重命名工具

一款基于 Wails v2 (Go + Vue 3) 的桌面级批量文件/文件夹重命名工具。

## ✨ 功能特性

- **双模式工作流**：选择工作目录 → 切换操作对象（子文件夹名 / 子文件名）
- **内置规则引擎**：
  - 添加前缀 / 后缀
  - 指定位置插入
  - 查找替换 / 正则替换
  - 自动编号（可排序后编号）
  - 根据文件名生成同名目录（支持移动/复制/仅建目录）
- **自定义脚本**：添加 `.bat` 脚本，接收原文件名为参数，输出新文件名
- **实时预览**：规则变化后自动防抖预览
- **排序支持**：按名称、修改时间、文件大小排序，编号与视觉顺序联动
- **脚本持久化**：添加的脚本下次启动自动加载

## 🖥️ 技术栈

| 层 | 技术 |
|---|------|
| 框架 | Wails v2 |
| 后端 | Go 1.21+ |
| 前端 | Vue 3 + TypeScript |
| UI 组件 | Element Plus |

## 🚀 开发

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式
wails dev

# 生产构建
wails build
```

## 📦 下载

前往 [Releases](https://github.com/lutherping/RenamerX/releases) 下载最新编译版本。

## 📄 License

MIT
