package sdrand

import (
	"math/rand"
	"strconv"
)

var (
	LowerLetters       = "abcdefghijklmnopqrstuvwxyz"
	UpperLetters       = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Letters            = LowerLetters + UpperLetters
	Numbers            = "0123456789"
	LettersNumbers     = Letters + Numbers
	LowerLetterNumbers = LowerLetters + Numbers
	UpperLetterNumbers = UpperLetters + Numbers
)

// String 生成指定长度的随机字符串
func String(n int, set string) string {
	set1 := []rune(set)
	nSet := len(set1)
	r := make([]rune, n)
	for i := range r {
		r[i] = set1[rand.Intn(nSet)]
	}
	return string(r)
}

// calculateLuhnChecksum 使用 Luhn 算法计算校验位
func calculateLuhnChecksum(s string) rune {
	// 将字符串转换为数字序列
	digits := make([]int, len(s))
	for i, c := range s {
		if c >= '0' && c <= '9' {
			digits[i] = int(c - '0')
		} else {
			// 对于非数字字符，使用其 ASCII 码的最后一位
			digits[i] = int(c) % 10
		}
	}

	// 从右向左，偶数位乘以2
	for i := len(digits) - 1; i >= 0; i -= 2 {
		digits[i] *= 2
		if digits[i] > 9 {
			digits[i] = digits[i]/10 + digits[i]%10
		}
	}

	// 计算所有数字的和
	sum := 0
	for _, d := range digits {
		sum += d
	}

	// 计算校验位
	checksum := (10 - (sum % 10)) % 10
	return rune('0' + checksum)
}

// StringWithChecksum 生成带校验位的随机字符串
// n 是期望的总长度，包括校验位
func StringWithChecksum(n int, set string) string {
	if n < 2 {
		return ""
	}
	// 生成 n-1 位的随机字符串
	baseStr := String(n-1, set)
	// 计算校验位
	checksum := calculateLuhnChecksum(baseStr)
	// 返回带校验位的字符串
	return baseStr + string(checksum)
}

// VerifyChecksum 验证字符串的校验位是否正确
func VerifyChecksum(s string) bool {
	if len(s) < 2 {
		return false
	}
	// 获取除最后一位外的字符串
	baseStr := s[:len(s)-1]
	// 获取校验位
	checksum := rune(s[len(s)-1])
	// 计算期望的校验位
	expectedChecksum := calculateLuhnChecksum(baseStr)
	// 比较校验位
	return checksum == expectedChecksum
}

// GenerateSecureString 生成安全的随机字符串（推荐使用）
// 参数说明：
// length: 总长度（包括校验位）
// useNumbers: 是否使用数字
// useLower: 是否使用小写字母
// useUpper: 是否使用大写字母
func GenerateSecureString(length int, useNumbers, useLower, useUpper bool) string {
	if length < 2 {
		return ""
	}

	// 构建字符集
	var charset string
	if useNumbers {
		charset += Numbers
	}
	if useLower {
		charset += LowerLetters
	}
	if useUpper {
		charset += UpperLetters
	}
	if charset == "" {
		charset = LettersNumbers // 默认使用所有字符
	}

	return StringWithChecksum(length, charset)
}

// GenerateNumericID 生成带校验位的数字ID
// length: 总长度（包括校验位）
func GenerateNumericID(length int) (int64, error) {
	if length < 2 {
		return 0, nil
	}

	// 生成数字字符串
	str := StringWithChecksum(length, Numbers)

	// 转换为int64
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// MustGenerateNumericID 生成带校验位的数字ID，如果出错则panic
// length: 总长度（包括校验位）
func MustGenerateNumericID(length int) int64 {
	id, err := GenerateNumericID(length)
	if err != nil {
		panic(err)
	}
	return id
}
