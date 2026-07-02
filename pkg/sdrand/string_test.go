package sdrand

import (
	"testing"
)

func TestStringWithChecksum(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		set      string
		expected bool
	}{
		{
			name:     "正常长度",
			length:   6,
			set:      Numbers,
			expected: true,
		},
		{
			name:     "使用字母和数字",
			length:   8,
			set:      LettersNumbers,
			expected: true,
		},
		{
			name:     "长度过短",
			length:   1,
			set:      Numbers,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringWithChecksum(tt.length, tt.set)
			if tt.expected {
				if len(got) != tt.length {
					t.Errorf("StringWithChecksum() length = %v, want %v", len(got), tt.length)
				}
				if !VerifyChecksum(got) {
					t.Errorf("StringWithChecksum() generated invalid checksum for %v", got)
				}
			} else {
				if got != "" {
					t.Errorf("StringWithChecksum() = %v, want empty string", got)
				}
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "有效校验位",
			input:    "1234567897", // 这是一个有效的 Luhn 校验数字
			expected: true,
		},
		{
			name:     "无效校验位",
			input:    "1234567890", // 这是一个无效的 Luhn 校验数字
			expected: false,
		},
		{
			name:     "空字符串",
			input:    "",
			expected: false,
		},
		{
			name:     "单字符",
			input:    "1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyChecksum(tt.input); got != tt.expected {
				t.Errorf("VerifyChecksum() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGenerateSecureString(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		useNumbers bool
		useLower   bool
		useUpper   bool
	}{
		{
			name:       "仅数字",
			length:     6,
			useNumbers: true,
			useLower:   false,
			useUpper:   false,
		},
		{
			name:       "数字和小写字母",
			length:     8,
			useNumbers: true,
			useLower:   true,
			useUpper:   false,
		},
		{
			name:       "所有字符",
			length:     10,
			useNumbers: true,
			useLower:   true,
			useUpper:   true,
		},
		{
			name:       "仅字母",
			length:     8,
			useNumbers: false,
			useLower:   true,
			useUpper:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSecureString(tt.length, tt.useNumbers, tt.useLower, tt.useUpper)
			if len(got) != tt.length {
				t.Errorf("GenerateSecureString() length = %v, want %v", len(got), tt.length)
			}
			if !VerifyChecksum(got) {
				t.Errorf("GenerateSecureString() generated invalid checksum for %v", got)
			}
		})
	}
}

func BenchmarkStringWithChecksum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		StringWithChecksum(10, LettersNumbers)
	}
}

func BenchmarkVerifyChecksum(b *testing.B) {
	str := StringWithChecksum(10, LettersNumbers)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyChecksum(str)
	}
}
