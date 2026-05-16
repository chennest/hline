package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/chennest/hline/internal/hash"
	"github.com/chennest/hline/internal/parse"
)

var version = "dev"

func main() {
	var afterN, beforeN int
	var nonFlagArgs []string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-A":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: -A requires a number")
				os.Exit(1)
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				fmt.Fprintf(os.Stderr, "Error: invalid -A value: %q\n", args[i])
				os.Exit(1)
			}
			afterN = n
		case args[i] == "-B":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: -B requires a number")
				os.Exit(1)
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				fmt.Fprintf(os.Stderr, "Error: invalid -B value: %q\n", args[i])
				os.Exit(1)
			}
			beforeN = n
		case args[i] == "-h" || args[i] == "--help":
			printUsage()
		case args[i] == "-v" || args[i] == "--version":
			fmt.Println("hcat", version)
			os.Exit(0)
		default:
			nonFlagArgs = append(nonFlagArgs, args[i])
		}
	}

	if len(nonFlagArgs) == 0 {
		printUsage()
	}

	filePath := nonFlagArgs[0]
	rangeArgs := nonFlagArgs[1:]

	// 解析范围
	r, err := parse.ParseRange(rangeArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// 按行分割
	lines := splitLines(string(data))

	// 计算实际范围
	start, end := resolveRange(r, len(lines))

	// -A/-B 配合单行号时，限制为单行 + 上下文
	if (afterN > 0 || beforeN > 0) && !r.HasOffset && r.End == 0 {
		end = start // 单行
	}

	// 应用 -B / -A 上下文
	if beforeN > 0 {
		start -= beforeN
		if start < 1 {
			start = 1
		}
	}
	if afterN > 0 {
		end += afterN
		if end > len(lines) {
			end = len(lines)
		}
	}

	// 输出带 hash 的行
	for i := start; i <= end; i++ {
		if i < 1 || i > len(lines) {
			continue
		}
		h := hash.Compute(lines[i-1])
		fmt.Printf("%d#%s|%s\n", i, h, lines[i-1])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: hcat [flags] <file> [range]")
	fmt.Fprintln(os.Stderr, "  flags:")
	fmt.Fprintln(os.Stderr, "    -A N          show N lines after target")
	fmt.Fprintln(os.Stderr, "    -B N          show N lines before target")
	fmt.Fprintln(os.Stderr, "    -h, --help    show this help")
	fmt.Fprintln(os.Stderr, "    -v, --version show version")
	fmt.Fprintln(os.Stderr, "  range: 5-10  |  5 +10  |  5")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  hcat file 5-10            # lines 5-10")
	fmt.Fprintln(os.Stderr, "  hcat file -A 3 -B 2 5    # line 5 + 2 before + 3 after")
	os.Exit(1)
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	content = trimRightNewlines(content)
	if content == "" {
		return nil
	}
	var lines []string
	for _, line := range split(content, '\n') {
		lines = append(lines, line)
	}
	return lines
}

func trimRightNewlines(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func split(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func resolveRange(r parse.Range, totalLines int) (int, int) {
	start := r.Start
	if start < 1 {
		start = 1
	}
	if start > totalLines {
		return totalLines + 1, totalLines
	}

	end := totalLines
	if r.HasOffset {
		end = start + r.Offset - 1
	} else if r.End > 0 {
		end = r.End
	}

	if end > totalLines {
		end = totalLines
	}

	return start, end
}
