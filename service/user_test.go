package service

import (
	"fmt"
	"testing"
)

func TestProcessUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "短用户名不处理",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "7-10位用户名使用3个星号",
			input:    "13812345678",
			expected: "1381***5678",
		},
		{
			name:     "邮箱地址处理",
			input:    "test@example.com",
			expected: "tes****@example.com",
		},
		{
			name:     "短邮箱地址处理",
			input:    "bb@example.com",
			expected: "bb***@example.com",
		},
		{
			name:     "超过10位用户名使用3个星号-偶数位",
			input:    "verylongusername",
			expected: "verylon***ername",
		},
		{
			name:     "超过10位用户名使用3个星号-奇数位",
			input:    "1234567",
			expected: "12***67",
		},
		{
			name:     "特殊字符处理",
			input:    "user.name@domain",
			expected: "use****@domain",
		},
	}

	fmt.Println("\n=== 用户名处理测试 ===")
	for _, tt := range tests {
		result := MaskUsername(tt.input)
		if result != tt.expected {
			fmt.Printf("\n❌ 测试失败: %s\n", tt.name)
			fmt.Printf("   输入: %s\n", tt.input)
			fmt.Printf("   实际: %s\n", result)
			fmt.Printf("   预期: %s\n", tt.expected)
			t.Errorf("测试失败: %s", tt.name)
		} else {
			fmt.Printf("\n✅ 测试通过: %s\n", tt.name)
			fmt.Printf("   输入: %s\n", tt.input)
			fmt.Printf("   输出: %s\n", result)
		}
	}
	fmt.Println("\n=== 测试完成 ===")
}
