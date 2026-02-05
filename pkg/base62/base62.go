package base62

import (
	"math"
	"strings"
)

// 62进制转换

// 数字+小写字母+大写字母

// 0-9 ： 0-9
// a-z : 10 - 35
// A-Z : 36 - 61
//type baseStr struct {
//	baseStr string
//}

var base62Str string

func MustInit(baseStr string) {
	if len(baseStr) == 0 {
		panic("进制字符串未初始化")
	}
	base62Str = baseStr
}

//const base62Str = `0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`

// Int2String 转化62进制
func Int2String(seq uint64) string {
	if seq == 0 {
		return string(base62Str[0])
	}
	var bl []byte
	for seq > 0 {
		mod := seq % 62
		div := seq / 62
		bl = append(bl, base62Str[mod])
		seq = div
	}
	// 反转得到的数
	return string(reverse(bl))
}

// reverse 反转
func reverse(s []byte) []byte {
	for i, j := 0, len(s)-1; i < len(s)/2; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// String2Int 62字符串转回10进制数
func String2Int(s string) uint64 {
	bl := []byte(s)
	bl = reverse(bl)
	var seq uint64
	for index, b := range bl {
		base := math.Pow(62, float64(index))
		seq += uint64(strings.Index(base62Str, string(b))) * uint64(base)
	}
	return seq

}
