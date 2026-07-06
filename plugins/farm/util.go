package main

import (
	"math"
	"time"
)

func now() int64 {
	return time.Now().Unix()
}

func computeLevel(exp int64) int {
	for i := 21; i > 0; i-- {
		need := (int64(math.Pow(float64(i), 4)) - 1) / 5
		if exp >= need {
			return i
		}
	}
	return 0
}

func fieldPrice(currentFieldCount int) int64 {
	base := float64(currentFieldCount)
	return int64(math.Pow(2.5, 0.75*base)*base) * 1000
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

func targetUserIDHash(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	return h
}

func (p *FarmPlugin) randomInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
