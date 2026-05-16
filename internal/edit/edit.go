package edit

import (
	"fmt"
	"os"
	"strings"

	"github.com/chennest/hline/internal/hash"
	"github.com/chennest/hline/internal/parse"
)

// Op 编辑操作类型
type Op string

const (
	OpReplace Op = "replace"
	OpAppend  Op = "append"
	OpPrepend Op = "prepend"
)

// Result 编辑结果
type Result struct {
	Success  bool
	Message  string
	Modified bool
}

// Edit 执行带 hash 校验的编辑操作
func Edit(filePath string, op Op, anchors []parse.Anchor, newContent string, dryRun bool) (Result, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Result{}, fmt.Errorf("read file: %w", err)
	}

	// 保留原始文件权限
	info, err := os.Stat(filePath)
	if err != nil {
		return Result{}, fmt.Errorf("stat file: %w", err)
	}
	mode := info.Mode()

	lines := splitLines(string(data))
	newLines := splitContent(newContent)

	switch op {
	case OpReplace:
		return doReplace(filePath, mode, lines, anchors, newLines, dryRun)
	case OpAppend:
		return doAppend(filePath, mode, lines, anchors, newLines, dryRun)
	case OpPrepend:
		return doPrepend(filePath, mode, lines, anchors, newLines, dryRun)
	default:
		return Result{}, fmt.Errorf("unknown operation: %s", op)
	}
}

// splitLines 按行分割文件内容，处理末尾换行
func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	// 去掉末尾换行后分割
	content = strings.TrimRight(content, "\n\r")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// splitContent 分割编辑内容，去掉末尾空行
func splitContent(content string) []string {
	content = strings.TrimRight(content, "\n\r")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// validateAnchor 校验锚点的 hash 是否匹配当前文件内容
func validateAnchor(lines []string, anchor parse.Anchor) (bool, string) {
	if anchor.Line < 1 || anchor.Line > len(lines) {
		return false, fmt.Sprintf("line %d out of range (file has %d lines)", anchor.Line, len(lines))
	}
	actualHash := hash.Compute(lines[anchor.Line-1])
	if actualHash != anchor.Hash {
		return false, fmt.Sprintf("hash mismatch at line %d\n  expected: %d#%s\n  current:  %d#%s|%s",
			anchor.Line, anchor.Line, anchor.Hash,
			anchor.Line, actualHash, lines[anchor.Line-1])
	}
	return true, ""
}

// writeFile 写入文件，确保末尾有换行
func writeFile(filePath string, mode os.FileMode, lines []string) error {
	output := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(filePath, []byte(output), mode)
}

func doReplace(filePath string, mode os.FileMode, lines []string, anchors []parse.Anchor, newLines []string, dryRun bool) (Result, error) {
	if len(anchors) < 1 {
		return Result{}, fmt.Errorf("replace requires at least one anchor")
	}

	startAnchor := anchors[0]
	endLine := startAnchor.Line
	if len(anchors) >= 2 {
		endLine = anchors[1].Line
	}

	// 校验起始锚点
	if ok, msg := validateAnchor(lines, startAnchor); !ok {
		return Result{Success: false, Message: msg}, nil
	}

	// 校验结束锚点（如果有范围）
	if len(anchors) >= 2 {
		if ok, msg := validateAnchor(lines, anchors[1]); !ok {
			return Result{Success: false, Message: msg}, nil
		}
	}

	if dryRun {
		return Result{Success: true, Message: "dry run: would apply replace", Modified: false}, nil
	}

	// 替换 lines[start..end] 为 newLines（1-based → 0-based）
	result := make([]string, 0, len(lines)-(endLine-startAnchor.Line+1)+len(newLines))
	result = append(result, lines[:startAnchor.Line-1]...)
	result = append(result, newLines...)
	result = append(result, lines[endLine:]...)

	if err := writeFile(filePath, mode, result); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	return Result{
		Success:  true,
		Modified: true,
		Message:  fmt.Sprintf("replaced lines %d-%d with %d line(s)", startAnchor.Line, endLine, len(newLines)),
	}, nil
}

func doAppend(filePath string, mode os.FileMode, lines []string, anchors []parse.Anchor, newLines []string, dryRun bool) (Result, error) {
	insertLine := len(lines) // 默认：文件末尾

	// 有锚点时，在该行之后插入
	if len(anchors) >= 1 && anchors[0].Line > 0 {
		if ok, msg := validateAnchor(lines, anchors[0]); !ok {
			return Result{Success: false, Message: msg}, nil
		}
		insertLine = anchors[0].Line
	}

	if dryRun {
		return Result{Success: true, Message: fmt.Sprintf("dry run: would append after line %d", insertLine), Modified: false}, nil
	}

	// 在 insertLine 之后插入（1-based → 0-based: 插入到 index insertLine 位置）
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:insertLine]...)
	result = append(result, newLines...)
	result = append(result, lines[insertLine:]...)

	if err := writeFile(filePath, mode, result); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	return Result{
		Success:  true,
		Modified: true,
		Message:  fmt.Sprintf("appended %d line(s) after line %d", len(newLines), insertLine),
	}, nil
}

func doPrepend(filePath string, mode os.FileMode, lines []string, anchors []parse.Anchor, newLines []string, dryRun bool) (Result, error) {
	insertLine := 0 // 默认：文件开头

	// 有锚点时，在该行之前插入
	if len(anchors) >= 1 && anchors[0].Line > 0 {
		if ok, msg := validateAnchor(lines, anchors[0]); !ok {
			return Result{Success: false, Message: msg}, nil
		}
		insertLine = anchors[0].Line - 1 // 0-based index
	}

	if dryRun {
		return Result{Success: true, Message: fmt.Sprintf("dry run: would prepend before line %d", insertLine+1), Modified: false}, nil
	}

	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:insertLine]...)
	result = append(result, newLines...)
	result = append(result, lines[insertLine:]...)

	if err := writeFile(filePath, mode, result); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	return Result{
		Success:  true,
		Modified: true,
		Message:  fmt.Sprintf("prepended %d line(s) before line %d", len(newLines), insertLine+1),
	}, nil
}
