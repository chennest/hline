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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: hsed <operation> [anchor [anchor2]]")
		fmt.Fprintln(os.Stderr, "  operation: replace | append | prepend")
		fmt.Fprintln(os.Stderr, "  anchor:    LINE#HASH  (e.g. 5#VKM)")
		fmt.Fprintln(os.Stderr, "  new content from stdin (heredoc)")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  hsed replace 5#VKM << 'EOF'")
		fmt.Fprintln(os.Stderr, "  hsed replace 5#VKM 10#ABC << 'EOF'")
		fmt.Fprintln(os.Stderr, "  hsed append 5#VKM << 'EOF'")
		fmt.Fprintln(os.Stderr, "  hsed prepend 5#VKM << 'EOF'")
		os.Exit(1)
	}

	// 解析操作类型
	opStr := strings.ToLower(os.Args[1])
	var op edit.Op
	switch opStr {
	case "replace":
		op = edit.OpReplace
	case "append":
		op = edit.OpAppend
	case "prepend":
		op = edit.OpPrepend
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown operation %q, expected replace|append|prepend\n", os.Args[1])
		os.Exit(1)
	}

	// 解析锚点
	anchorArgs := os.Args[2:]
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

	// 从 stdin 读取新内容
	newContent, err := readStdin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
		os.Exit(1)
	}

	// 查找文件路径：从环境变量或锚点推导
	// hsed 通过 stdin 接收内容，文件路径需要额外参数
	// 使用 --file 参数或 HLINE_FILE 环境变量
	filePath := os.Getenv("HLINE_FILE")
	if filePath == "" {
		// 从参数中查找 --file
		for i, arg := range os.Args {
			if arg == "--file" && i+1 < len(os.Args) {
				filePath = os.Args[i+1]
				break
			}
			if strings.HasPrefix(arg, "--file=") {
				filePath = strings.TrimPrefix(arg, "--file=")
				break
			}
		}
	}
	if filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: file path required")
		fmt.Fprintln(os.Stderr, "  Set HLINE_FILE env var or use --file <path>")
		os.Exit(1)
	}

	// 检查是否有 -p/--preview 标志
	dryRun := false
	for _, arg := range os.Args {
		if arg == "-p" || arg == "--preview" {
			dryRun = true
		}
	}

	// 执行编辑
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
