package pkg

import "strings"

// Contains 字典中包含意图字符
func Contains(target string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(keyword, target) {
			return true
		}
	}
	return false
}
