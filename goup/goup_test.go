package goup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPv4Validate(t *testing.T) {
	tests := []struct {
		name string
		isv4 bool
	}{
		{name: "8.8.8.8", isv4: true},
		{name: "255.255.255.255", isv4: true},
		{name: "1.31.123.12", isv4: true},
		{name: "1.31.123.", isv4: false},
		{name: "1.31.123.5555", isv4: false},
		{name: "1.31.123.123:8080", isv4: false},
		{name: "812379123.int", isv4: false},
		{name: "test.812379123.int", isv4: false},
		{name: "[fe80::12]", isv4: false},
		{name: "a.b.c.d.e.1.3.4.cn-northwest-1.eb.amazonaws.com.cn", isv4: false},
		{name: "123.高知.jp", isv4: false},
		{name: "111.123.高知.jp", isv4: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.isv4, isIPv4(test.name))
		})
	}
}

func BenchmarkIPv4Validate(b *testing.B) {
	b.Run("positive-0", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isIPv4("8.8.8.8")
		}
	})
	b.Run("negative-1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isIPv4("123.高知.jp")
		}
	})
	b.Run("negative-2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isIPv4("111.123.高知.jp")
		}
	})
	b.Run("negative-3", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isIPv4("a.b.c.d.e.1.3.4.cn-northwest-1.eb.amazonaws.com.cn")
		}
	})
}
