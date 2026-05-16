package parse

import (
	"fmt"
	"strconv"
	"strings"
)

// Anchor 表示解析后的 行号#HASH 锚点
type Anchor struct {
	Line int
	Hash string
}

// ParseAnchor 解析 "5#VKM" 格式的锚点
func ParseAnchor(s string) (Anchor, error) {
	parts := strings.SplitN(s, "#", 2)
	if len(parts) != 2 {
		return Anchor{}, fmt.Errorf("invalid anchor format: %q, expected LINE#HASH", s)
	}

	line, err := strconv.Atoi(parts[0])
	if err != nil || line < 1 {
		return Anchor{}, fmt.Errorf("invalid line number: %q", parts[0])
	}

	hash := strings.ToUpper(parts[1])
	if len(hash) != 2 {
		return Anchor{}, fmt.Errorf("hash must be 2 letters (A-Z), got %d chars: %q", len(hash), hash)
	}
	for _, c := range hash {
		if c < 'A' || c > 'Z' {
			return Anchor{}, fmt.Errorf("hash must be A-Z, got invalid char %q", c)
		}
	}

	return Anchor{Line: line, Hash: hash}, nil
}

// Range 表示解析后的行范围
type Range struct {
	Start     int  // 起始行号（1-based）
	End       int  // 结束行号（0 表示到文件末尾）
	Offset    int  // +N 语法时的偏移量
	HasOffset bool // 是否使用 +N 语法
}

// ParseRange 解析范围参数："" | "5" | "5-10" | "5 +10"
func ParseRange(args []string) (Range, error) {
	if len(args) == 0 {
		return Range{Start: 1}, nil
	}

	// 支持 "5-10" 单参数格式
	if len(args) == 1 && strings.Contains(args[0], "-") {
		parts := strings.SplitN(args[0], "-", 2)
		s, err := strconv.Atoi(parts[0])
		if err != nil || s < 1 {
			return Range{}, fmt.Errorf("invalid start line: %q", args[0])
		}
		e, err := strconv.Atoi(parts[1])
		if err != nil || e < 1 {
			return Range{}, fmt.Errorf("invalid end line: %q", args[0])
		}
		if e < s {
			return Range{}, fmt.Errorf("end line %d must >= start line %d", e, s)
		}
		return Range{Start: s, End: e}, nil
	}

	start, err := strconv.Atoi(args[0])
	if err != nil || start < 1 {
		return Range{}, fmt.Errorf("invalid start line: %q", args[0])
	}

	if len(args) == 1 {
		return Range{Start: start}, nil
	}

	// +N 语法：从 start 开始看 N 行
	if strings.HasPrefix(args[1], "+") {
		offset, err := strconv.Atoi(args[1][1:])
		if err != nil || offset < 1 {
			return Range{}, fmt.Errorf("invalid offset: %q", args[1])
		}
		return Range{Start: start, Offset: offset, HasOffset: true}, nil
	}

	// N-M 语法
	end, err := strconv.Atoi(args[1])
	if err != nil || end < 1 {
		return Range{}, fmt.Errorf("invalid end line: %q", args[1])
	}
	if end < start {
		return Range{}, fmt.Errorf("end line %d must >= start line %d", end, start)
	}

	return Range{Start: start, End: end}, nil
}
