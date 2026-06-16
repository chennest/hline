package edit

import (
	"fmt"
	"os"
	"sort"
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
	OpBatch   Op = "batch"
)

// BatchEntry 表示批量操作中的单个条目
type BatchEntry struct {
	Op      Op
	Anchors []parse.Anchor
	Content string
}

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
	lineEnding := detectLineEnding(string(data))

	lines := splitLines(string(data))
	newLines := splitContent(newContent)

	switch op {
	case OpReplace:
		return doReplace(filePath, mode, lineEnding, lines, anchors, newLines, dryRun)
	case OpAppend:
		return doAppend(filePath, mode, lineEnding, lines, anchors, newLines, dryRun)
	case OpPrepend:
		return doPrepend(filePath, mode, lineEnding, lines, anchors, newLines, dryRun)
	default:
		return Result{}, fmt.Errorf("unknown operation: %s", op)
	}
}

// splitLines 必须与 cmd/hcat 的 splitLines 保持一致：两者算 hash 的行内容定义相同。
// 剥离所有换行符（\r\n、孤立 \r、\n）以确保 CRLF 文件也能正确往返。
func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "\n")
}

// splitContent 用与 splitLines 相同的规则处理用户 stdin content。
func splitContent(content string) []string {
	return splitLines(content)
}

// detectLineEnding 探测原文件换行符；writeFile 按此写回，避免 mixed endings。
func detectLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
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

// writeFile 写入文件，按 lineEnding 串接行并保证末尾换行。
func writeFile(filePath string, mode os.FileMode, lines []string, lineEnding string) error {
	output := strings.Join(lines, lineEnding) + lineEnding
	return os.WriteFile(filePath, []byte(output), mode)
}

func doReplace(filePath string, mode os.FileMode, lineEnding string, lines []string, anchors []parse.Anchor, newLines []string, dryRun bool) (Result, error) {
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

	if err := writeFile(filePath, mode, result, lineEnding); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	return Result{
		Success:  true,
		Modified: true,
		Message:  fmt.Sprintf("replaced lines %d-%d with %d line(s)", startAnchor.Line, endLine, len(newLines)),
	}, nil
}

func doAppend(filePath string, mode os.FileMode, lineEnding string, lines []string, anchors []parse.Anchor, newLines []string, dryRun bool) (Result, error) {
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

	if err := writeFile(filePath, mode, result, lineEnding); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	return Result{
		Success:  true,
		Modified: true,
		Message:  fmt.Sprintf("appended %d line(s) after line %d", len(newLines), insertLine),
	}, nil
}

func doPrepend(filePath string, mode os.FileMode, lineEnding string, lines []string, anchors []parse.Anchor, newLines []string, dryRun bool) (Result, error) {
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

	if err := writeFile(filePath, mode, result, lineEnding); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	return Result{
		Success:  true,
		Modified: true,
		Message:  fmt.Sprintf("prepended %d line(s) before line %d", len(newLines), insertLine+1),
	}, nil
}

// Batch 原子性批量应用多个编辑操作
func Batch(filePath string, entries []BatchEntry, dryRun bool) (Result, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Result{}, fmt.Errorf("read file: %w", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return Result{}, fmt.Errorf("stat file: %w", err)
	}
	mode := info.Mode()
	lineEnding := detectLineEnding(string(data))

	lines := splitLines(string(data))
	totalLines := len(lines)

	for i, entry := range entries {
		for _, anchor := range entry.Anchors {
			if anchor.Line < 1 || anchor.Line > totalLines {
				return Result{Success: false, Message: fmt.Sprintf("entry %d: line %d out of range (file has %d lines)", i+1, anchor.Line, totalLines)}, nil
			}
			if ok, msg := validateAnchor(lines, anchor); !ok {
				return Result{Success: false, Message: fmt.Sprintf("entry %d: %s", i+1, msg)}, nil
			}
		}
		if entry.Op == OpReplace && len(entry.Anchors) >= 2 {
			if entry.Anchors[0].Line > entry.Anchors[1].Line {
				return Result{Success: false, Message: fmt.Sprintf("entry %d: replace range descending: start=%d, end=%d", i+1, entry.Anchors[0].Line, entry.Anchors[1].Line)}, nil
			}
		}
	}

	type replaceRange struct {
		start, end int
		entryIdx   int
	}
	var replaces []replaceRange
	for i, entry := range entries {
		if entry.Op == OpReplace {
			s := entry.Anchors[0].Line
			e := s
			if len(entry.Anchors) >= 2 {
				e = entry.Anchors[1].Line
			}
			replaces = append(replaces, replaceRange{start: s, end: e, entryIdx: i})
		}
	}

	for i := 0; i < len(replaces); i++ {
		for j := i + 1; j < len(replaces); j++ {
			a, b := replaces[i], replaces[j]
			if max(a.start, b.start) <= min(a.end, b.end) {
				return Result{Success: false, Message: fmt.Sprintf("conflict: replace[%d..%d] overlaps replace[%d..%d]", a.start, a.end, b.start, b.end)}, nil
			}
		}
	}

	for _, entry := range entries {
		if entry.Op == OpAppend || entry.Op == OpPrepend {
			if len(entry.Anchors) > 0 {
				x := entry.Anchors[0].Line
				for _, r := range replaces {
					if x >= r.start && x <= r.end {
						return Result{Success: false, Message: fmt.Sprintf("conflict: %s@%d falls inside replace[%d..%d]", entry.Op, x, r.start, r.end)}, nil
					}
				}
			}
		}
	}

	type indexedEntry struct {
		entry BatchEntry
		idx   int
		key   int
	}
	indexed := make([]indexedEntry, len(entries))
	for i, entry := range entries {
		var key int
		switch entry.Op {
		case OpReplace:
			key = entry.Anchors[0].Line
		case OpAppend:
			if len(entry.Anchors) > 0 {
				key = entry.Anchors[0].Line
			} else {
				key = totalLines + 1
			}
		case OpPrepend:
			if len(entry.Anchors) > 0 {
				key = entry.Anchors[0].Line - 1
			} else {
				key = 0
			}
		}
		indexed[i] = indexedEntry{entry: entry, idx: i, key: key}
	}

	sort.SliceStable(indexed, func(i, j int) bool {
		if indexed[i].key != indexed[j].key {
			return indexed[i].key > indexed[j].key
		}
		// 同 key 时 idx 降序：让先输入的 entry 后应用，使其更靠近锚点
		return indexed[i].idx > indexed[j].idx
	})

	if dryRun {
		for i, e := range entries {
			newLines := splitContent(e.Content)
			desc := describeEntry(e, totalLines)
			fmt.Printf("[%d] %s -> %d line(s)\n", i+1, desc, len(newLines))
		}
		return Result{
			Success:  true,
			Modified: false,
			Message:  fmt.Sprintf("dry run: would apply %d change(s), file unchanged", len(entries)),
		}, nil
	}

	resultLines := make([]string, len(lines))
	copy(resultLines, lines)

	for _, ie := range indexed {
		resultLines = applyOne(resultLines, ie.entry, totalLines)
	}

	if err := writeFile(filePath, mode, resultLines, lineEnding); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	var parts []string
	for _, e := range entries {
		switch e.Op {
		case OpReplace:
			s := e.Anchors[0].Line
			eLine := s
			if len(e.Anchors) >= 2 {
				eLine = e.Anchors[1].Line
			}
			newLines := splitContent(e.Content)
			parts = append(parts, fmt.Sprintf("replaced lines %d-%d with %d line(s)", s, eLine, len(newLines)))
		case OpAppend:
			newLines := splitContent(e.Content)
			if len(e.Anchors) > 0 {
				parts = append(parts, fmt.Sprintf("appended %d line(s) after line %d", len(newLines), e.Anchors[0].Line))
			} else {
				parts = append(parts, fmt.Sprintf("appended %d line(s) at end", len(newLines)))
			}
		case OpPrepend:
			newLines := splitContent(e.Content)
			if len(e.Anchors) > 0 {
				parts = append(parts, fmt.Sprintf("prepended %d line(s) before line %d", len(newLines), e.Anchors[0].Line))
			} else {
				parts = append(parts, fmt.Sprintf("prepended %d line(s) at start", len(newLines)))
			}
		}
	}

	return Result{
		Success:  true,
		Modified: true,
		Message:  fmt.Sprintf("applied %d change(s): %s", len(entries), strings.Join(parts, ", ")),
	}, nil
}

// applyOne 应用单个 entry 到 lines，返回新切片（不写文件）。
// totalLines 为原始文件行数；无锚点 append 必须用 totalLines 而非 len(lines)，
// 否则多个无锚点 append 会因 lines 增长而顺序反转。
func applyOne(lines []string, entry BatchEntry, totalLines int) []string {
	newLines := splitContent(entry.Content)
	switch entry.Op {
	case OpReplace:
		start := entry.Anchors[0].Line
		end := start
		if len(entry.Anchors) >= 2 {
			end = entry.Anchors[1].Line
		}
		result := make([]string, 0, len(lines)-(end-start+1)+len(newLines))
		result = append(result, lines[:start-1]...)
		result = append(result, newLines...)
		result = append(result, lines[end:]...)
		return result
	case OpAppend:
		insertLine := totalLines
		if len(entry.Anchors) > 0 {
			insertLine = entry.Anchors[0].Line
		}
		result := make([]string, 0, len(lines)+len(newLines))
		result = append(result, lines[:insertLine]...)
		result = append(result, newLines...)
		result = append(result, lines[insertLine:]...)
		return result
	case OpPrepend:
		insertLine := 0
		if len(entry.Anchors) > 0 {
			insertLine = entry.Anchors[0].Line - 1
		}
		result := make([]string, 0, len(lines)+len(newLines))
		result = append(result, lines[:insertLine]...)
		result = append(result, newLines...)
		result = append(result, lines[insertLine:]...)
		return result
	default:
		return lines
	}
}

// describeEntry 生成 entry 的人类可读描述（用于 dry run 预览）
func describeEntry(entry BatchEntry, totalLines int) string {
	switch entry.Op {
	case OpReplace:
		s := entry.Anchors[0].Line
		e := s
		if len(entry.Anchors) >= 2 {
			e = entry.Anchors[1].Line
		}
		if s == e {
			return fmt.Sprintf("replace %d#%s", s, entry.Anchors[0].Hash)
		}
		return fmt.Sprintf("replace %d#%s %d#%s", s, entry.Anchors[0].Hash, e, entry.Anchors[1].Hash)
	case OpAppend:
		if len(entry.Anchors) > 0 {
			return fmt.Sprintf("append %d#%s", entry.Anchors[0].Line, entry.Anchors[0].Hash)
		}
		return "append (end)"
	case OpPrepend:
		if len(entry.Anchors) > 0 {
			return fmt.Sprintf("prepend %d#%s", entry.Anchors[0].Line, entry.Anchors[0].Hash)
		}
		return "prepend (start)"
	default:
		return string(entry.Op)
	}
}
