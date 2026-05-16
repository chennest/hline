package hash

import (
	"github.com/zeebo/xxh3"
)

// alphabet 用于将 hash 值映射为可读字母
const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Compute 计算行内容的 2 位字母 hash（A-Z）
// 使用 xxhash3，仅基于行内容，不绑定行号
func Compute(content string) string {
	h := xxh3.HashString(content)
	return string(alphabet[h%26]) +
		string(alphabet[(h/26)%26])
}
