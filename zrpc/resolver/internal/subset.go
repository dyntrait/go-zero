package internal

import "math/rand"

// 从一个给定的字符串切片 set 中随机选择 sub 个不重复的字符串
func subset(set []string, sub int) []string {
	rand.Shuffle(len(set), func(i, j int) { //随机打乱
		set[i], set[j] = set[j], set[i]
	})
	if len(set) <= sub {
		return set
	}

	return set[:sub]
}
