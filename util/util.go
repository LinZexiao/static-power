package util

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/go-state-types/big"
)

func DiffSet[K comparable, V any](before, after map[K]V) (add, keep, rm map[K]V) {
	add = make(map[K]V)
	keep = make(map[K]V)
	rm = make(map[K]V)

	for k, vb := range before {
		if va, ok := after[k]; ok {
			keep[k] = va
		} else {
			rm[k] = vb
		}
	}
	for k, v := range after {
		if _, ok := before[k]; !ok {
			add[k] = v
		}
	}
	return
}

func PiB(s string) float64 {
	TiB := big.NewInt(1024 * 1024 * 1024 * 1024)

	if s == "" {
		return 0
	}
	bp, err := big.FromString(s)
	if err != nil {
		panic(fmt.Errorf("parse big int %s: %w", s, err))
	}
	return float64(big.Div(bp, TiB).Int64()) / 1024
}

func SliceMap[T, U any](s []T, f func(T) U) []U {
	var ret []U
	for _, v := range s {
		ret = append(ret, f(v))
	}
	return ret
}

func SliceFilter[T any](s []T, f func(T) bool) []T {
	var ret []T
	for _, v := range s {
		if f(v) {
			ret = append(ret, v)
		}
	}
	return ret
}

func Unique[T comparable](s []T) []T {
	m := make(map[T]struct{})
	var ret []T
	for _, v := range s {
		if _, ok := m[v]; !ok {
			m[v] = struct{}{}
			ret = append(ret, v)
		}
	}
	return ret
}

func IsVenus(s string) bool {
	return strings.Contains(s, "venus") || strings.Contains(s, "droplet")
}

func IsLotus(s string) bool {
	return strings.Contains(s, "lotus") || strings.Contains(s, "boost")
}

func Slice2Map[K comparable, T any](s []T, key func(T) K) map[K]T {
	m := make(map[K]T)
	for _, v := range s {
		m[key(v)] = v
	}
	return m
}
