package main

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertAmountToValue(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		tokenDecimals int
		expected      string
		description   string
	}{
		{
			name:          "1 ETH with 18 decimals",
			amount:        1.0,
			tokenDecimals: 18,
			expected:      "1000000000000000000",
			description:   "1 ETH = 10^18 wei",
		},
		{
			name:          "0.5 ETH with 18 decimals",
			amount:        0.5,
			tokenDecimals: 18,
			expected:      "500000000000000000",
			description:   "0.5 ETH = 5 * 10^17 wei",
		},
		{
			name:          "0.1 ETH with 18 decimals",
			amount:        0.1,
			tokenDecimals: 18,
			expected:      "100000000000000000",
			description:   "0.1 ETH = 10^17 wei",
		},
		{
			name:          "1 USDT with 6 decimals",
			amount:        1.0,
			tokenDecimals: 6,
			expected:      "1000000",
			description:   "1 USDT = 10^6 smallest unit",
		},
		{
			name:          "0.5 USDT with 6 decimals",
			amount:        0.5,
			tokenDecimals: 6,
			expected:      "500000",
			description:   "0.5 USDT = 5 * 10^5 smallest unit",
		},
		{
			name:          "1 BTC with 8 decimals",
			amount:        1.0,
			tokenDecimals: 8,
			expected:      "100000000",
			description:   "1 BTC = 10^8 satoshi",
		},
		{
			name:          "0.00000001 BTC with 8 decimals",
			amount:        0.00000001,
			tokenDecimals: 8,
			expected:      "1",
			description:   "最小单位 1 satoshi",
		},
		{
			name:          "zero amount",
			amount:        0.0,
			tokenDecimals: 18,
			expected:      "0",
			description:   "零金额应该返回 0",
		},
		{
			name:          "very small amount with 18 decimals",
			amount:        0.000000000000000001,
			tokenDecimals: 18,
			expected:      "1",
			description:   "最小单位 1 wei",
		},
		{
			name:          "large amount with 18 decimals",
			amount:        1000000.0,
			tokenDecimals: 18,
			expected:      "1000000000000000000000000",
			description:   "100万 ETH = 10^6 * 10^18 wei",
		},
		{
			name:          "amount with 0 decimals",
			amount:        123.456,
			tokenDecimals: 0,
			expected:      "123",
			description:   "0 位小数，应该向下取整",
		},
		{
			name:          "amount with 1 decimal",
			amount:        12.34,
			tokenDecimals: 1,
			expected:      "123",
			description:   "1 位小数，12.34 * 10 = 123.4，向下取整为 123",
		},
		{
			name:          "negative amount (edge case)",
			amount:        -1.0,
			tokenDecimals: 18,
			expected:      "-1000000000000000000",
			description:   "负数金额（虽然实际业务可能不允许，但测试函数行为）",
		},
		{
			name:          "very large decimals",
			amount:        1.0,
			tokenDecimals: 30,
			expected:      "1000000000000000000000000000000",
			description:   "非常大的小数位数",
		},
		{
			name:          "fractional amount with 6 decimals",
			amount:        0.123456,
			tokenDecimals: 6,
			expected:      "123456",
			description:   "0.123456 USDT = 123456 smallest unit",
		},
		{
			name:          "amount with many decimal places",
			amount:        0.000001,
			tokenDecimals: 18,
			expected:      "1000000000000",
			description:   "0.000001 ETH = 10^12 wei",
		},
		{
			name:          "amount with 4.18",
			amount:        4.18,
			tokenDecimals: 6,
			expected:      "4180000",
			description:   "4.18 USDT = 4180000 smallest unit",
		},
		{
			name:          "amount with 9.18",
			amount:        9.18,
			tokenDecimals: 6,
			expected:      "9180000",
			description:   "9.18 USDT = 9180000 smallest unit",
		},
		{
			name:          "amount with 39.18",
			amount:        39.18,
			tokenDecimals: 6,
			expected:      "39180000",
			description:   "39.18 USDT = 39180000 smallest unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertAmountToValue(tt.amount, tt.tokenDecimals)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestConvertAmountToValue_EdgeCases 测试边界情况
func TestConvertAmountToValue_EdgeCases(t *testing.T) {
	// 测试最大 float64 值
	t.Run("max float64", func(t *testing.T) {
		result := ConvertAmountToValue(math.MaxFloat64, 18)
		// 结果应该是一个非常大的数字字符串
		assert.NotEmpty(t, result)
		assert.Greater(t, len(result), 0)
	})

	// 测试非常小的正数
	t.Run("very small positive", func(t *testing.T) {
		result := ConvertAmountToValue(1e-18, 18)
		// 1e-18 * 10^18 = 1
		assert.Equal(t, "1", result)
	})

	// 测试小于最小单位的金额（应该向下取整为 0）
	t.Run("amount smaller than smallest unit", func(t *testing.T) {
		result := ConvertAmountToValue(0.5e-18, 18)
		// 0.5e-18 * 10^18 = 0.5，向下取整为 0
		assert.Equal(t, "0", result)
	})

	// 测试精确的整数转换
	t.Run("exact integer conversion", func(t *testing.T) {
		result := ConvertAmountToValue(100.0, 18)
		assert.Equal(t, "100000000000000000000", result)
	})
}

// TestConvertAmountToValue_Precision 测试精度问题
func TestConvertAmountToValue_Precision(t *testing.T) {
	// 测试浮点数精度问题
	// 0.1 + 0.2 在浮点数中不等于 0.3
	t.Run("floating point precision", func(t *testing.T) {
		amount := 0.1 + 0.2
		result := ConvertAmountToValue(amount, 18)
		// 0.3 * 10^18 = 300000000000000000
		assert.Equal(t, "300000000000000000", result)
	})

	// 测试多个小数位的精度
	t.Run("multiple decimal places", func(t *testing.T) {
		result := ConvertAmountToValue(1.999999999999999999, 18)
		// 应该向下取整
		assert.NotEmpty(t, result)
	})
}
