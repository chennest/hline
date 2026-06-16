package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chennest/hline/internal/edit"
	"github.com/chennest/hline/internal/parse"
)

var version = "dev"

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage()
		}
		if arg == "-v" || arg == "--version" {
			fmt.Println("hsed", version)
			os.Exit(0)
		}
	}

	dryRun := false
	var positional []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-p", "--preview":
			dryRun = true
		case "-h", "--help", "-v", "--version":
		default:
			positional = append(positional, arg)
		}
	}

	if len(positional) < 2 {
		printUsage()
	}

	filePath := positional[0]
	opStr := strings.ToLower(positional[1])

	var op edit.Op
	switch opStr {
	case "replace":
		op = edit.OpReplace
	case "append":
		op = edit.OpAppend
	case "prepend":
		op = edit.OpPrepend
	case "batch":
		op = edit.OpBatch
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown operation %q, expected replace|append|prepend|batch\n", positional[1])
		os.Exit(1)
	}

	if op == edit.OpBatch {
		stdinContent, err := readStdin()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
			os.Exit(1)
		}
		entries, err := parseBatch(stdinContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		result, err := edit.Batch(filePath, entries, dryRun)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		if result.Success {
			fmt.Println(result.Message)
		} else {
			fmt.Fprintln(os.Stderr, "ERROR:", result.Message)
			os.Exit(1)
		}
		return
	}

	anchorArgs := positional[2:]
	if len(anchorArgs) == 0 && op == edit.OpReplace {
		fmt.Fprintln(os.Stderr, "Error: replace requires at least one anchor")
		os.Exit(1)
	}

	var anchors []parse.Anchor
	for _, arg := range anchorArgs {
		anchor, err := parse.ParseAnchor(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		anchors = append(anchors, anchor)
	}

	newContent, err := readStdin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
		os.Exit(1)
	}

	result, err := edit.Edit(filePath, op, anchors, newContent, dryRun)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if result.Success {
		fmt.Println(result.Message)
	} else {
		fmt.Fprintln(os.Stderr, "ERROR:", result.Message)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: hsed [flags] <file> <operation> [anchor [anchor2]]")
	fmt.Fprintln(os.Stderr, "  flags:")
	fmt.Fprintln(os.Stderr, "    -h, --help    show this help")
	fmt.Fprintln(os.Stderr, "    -v, --version show version")
	fmt.Fprintln(os.Stderr, "    -p, --preview dry run, show diff only (any position)")
	fmt.Fprintln(os.Stderr, "  operation: replace | append | prepend | batch")
	fmt.Fprintln(os.Stderr, "  anchor:    LINE#HASH  (e.g. 5#VK)")
	fmt.Fprintln(os.Stderr, "  new content from stdin (heredoc)")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf replace 5#VK << 'EOF'")
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf replace 5#VK 10#AB << 'EOF'")
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf append 5#VK << 'EOF'")
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf prepend 5#VK << 'EOF'")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  # batch: multiple changes in one call (atomic, all-or-nothing)")
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf batch << 'EOF'")
	fmt.Fprintln(os.Stderr, "  replace 5#VK 6#XJ")
	fmt.Fprintln(os.Stderr, "  new line 1")
	fmt.Fprintln(os.Stderr, "  new line 2")
	fmt.Fprintln(os.Stderr, "  ---")
	fmt.Fprintln(os.Stderr, "  append 10#AB")
	fmt.Fprintln(os.Stderr, "  appended line")
	fmt.Fprintln(os.Stderr, "  EOF")
	os.Exit(1)
}

func readStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// 没有管道输入，内容为空（用于删除行）
		return "", nil
	}

	var buf strings.Builder
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		buf.WriteString(line)
		if err == io.EOF {
			break
		}
	}
	return buf.String(), nil
}

// parseBatch 解析 batch 模式的 stdin 多段格式。
// 段之间用独占一行的 "---" 分隔。每段第一行是操作头（op + anchors），
// 后续行是该段的 content。末尾的 "---" 可选（产生空末段，自动忽略）。
// 中间空段（连续 "---" 或起始即 "---"）视为错误。
func parseBatch(stdin string) ([]edit.BatchEntry, error) {
	stdin = strings.TrimRight(stdin, "\n\r")
	if stdin == "" {
		return nil, fmt.Errorf("empty stdin")
	}

	allLines := strings.Split(stdin, "\n")

	// 按独占一行的 "---" 切段
	var rawSegments [][]string
	var current []string
	for _, line := range allLines {
		if line == "---" {
			rawSegments = append(rawSegments, current)
			current = nil
		} else {
			current = append(current, line)
		}
	}
	rawSegments = append(rawSegments, current)

	// 中间空段非法；末尾空段（用户在最后加了 "---"）允许并跳过
	for i, seg := range rawSegments {
		if len(seg) == 0 && i != len(rawSegments)-1 {
			return nil, fmt.Errorf("empty segment %d (consecutive \"---\"?)", i+1)
		}
	}

	var entries []edit.BatchEntry
	for i, seg := range rawSegments {
		if len(seg) == 0 {
			continue
		}

		header := strings.Fields(seg[0])
		if len(header) < 1 {
			return nil, fmt.Errorf("segment %d: missing operation", i+1)
		}

		opStr := strings.ToLower(header[0])
		var op edit.Op
		switch opStr {
		case "replace":
			op = edit.OpReplace
		case "append":
			op = edit.OpAppend
		case "prepend":
			op = edit.OpPrepend
		default:
			return nil, fmt.Errorf("segment %d: unknown operation %q", i+1, header[0])
		}

		var anchors []parse.Anchor
		for _, a := range header[1:] {
			anchor, err := parse.ParseAnchor(a)
			if err != nil {
				return nil, fmt.Errorf("segment %d: %w", i+1, err)
			}
			anchors = append(anchors, anchor)
		}

		if op == edit.OpReplace && len(anchors) < 1 {
			return nil, fmt.Errorf("segment %d: replace requires at least one anchor", i+1)
		}
		maxAnchors := 2
		if op == edit.OpAppend || op == edit.OpPrepend {
			maxAnchors = 1
		}
		if len(anchors) > maxAnchors {
			return nil, fmt.Errorf("segment %d: %s accepts at most %d anchor(s), got %d", i+1, opStr, maxAnchors, len(anchors))
		}

		content := ""
		if len(seg) > 1 {
			content = strings.Join(seg[1:], "\n")
		}

		entries = append(entries, edit.BatchEntry{
			Op:      op,
			Anchors: anchors,
			Content: content,
		})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no segments found")
	}
	return entries, nil
}
