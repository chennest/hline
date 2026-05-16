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
	if len(os.Args) < 2 {
		printUsage()
	}

	// 全局标志
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage()
		}
		if arg == "-v" || arg == "--version" {
			fmt.Println("hsed", version)
			os.Exit(0)
		}
	}

	// 解析文件路径（第一个参数）
	if len(os.Args) < 2 {
		printUsage()
	}
	filePath := os.Args[1]

	// 解析操作类型（第二个参数）
	if len(os.Args) < 3 {
		printUsage()
	}
	opStr := strings.ToLower(os.Args[2])
	var op edit.Op
	switch opStr {
	case "replace":
		op = edit.OpReplace
	case "append":
		op = edit.OpAppend
	case "prepend":
		op = edit.OpPrepend
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown operation %q, expected replace|append|prepend\n", os.Args[2])
		os.Exit(1)
	}

	// 解析锚点（第三个参数起）
	anchorArgs := os.Args[3:]
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

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: hsed [flags] <file> <operation> [anchor [anchor2]]")
	fmt.Fprintln(os.Stderr, "  flags:")
	fmt.Fprintln(os.Stderr, "    -h, --help    show this help")
	fmt.Fprintln(os.Stderr, "    -v, --version show version")
	fmt.Fprintln(os.Stderr, "    -p, --preview dry run, show diff only")
	fmt.Fprintln(os.Stderr, "  operation: replace | append | prepend")
	fmt.Fprintln(os.Stderr, "  anchor:    LINE#HASH  (e.g. 5#VK)")
	fmt.Fprintln(os.Stderr, "  new content from stdin (heredoc)")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf replace 5#VK << 'EOF'")
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf replace 5#VK 10#AB << 'EOF'")
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf append 5#VK << 'EOF'")
	fmt.Fprintln(os.Stderr, "  hsed /etc/nginx/nginx.conf prepend 5#VK << 'EOF'")
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
