package service

import (
	"regexp"
	"strings"
)

// 验证版本号格式
func IsValidVersion(version string) bool {
	// 版本号格式：x.y.z，其中 x、y、z 都是数字
	pattern := `^\d+\.\d+\.\d+$`
	match, _ := regexp.MatchString(pattern, version)
	return match
}

// 比较版本号
// 返回值：1 表示 version1 > version2，-1 表示 version1 < version2，0 表示相等
func CompareVersion(version1, version2 string) int {
	v1 := strings.Split(version1, ".")
	v2 := strings.Split(version2, ".")

	for i := 0; i < 3; i++ {
		if i >= len(v1) || i >= len(v2) {
			return 0
		}
		if v1[i] > v2[i] {
			return 1
		}
		if v1[i] < v2[i] {
			return -1
		}
	}
	return 0
}

// 检查是否需要更新
func NeedUpdate(currentVersion, minVersion string) bool {
	if currentVersion == "" || minVersion == "" {
		return false
	}
	return CompareVersion(currentVersion, minVersion) < 0
}
