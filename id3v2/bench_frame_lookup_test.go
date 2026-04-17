package id3v2

import (
	"fmt"
	"testing"
)

func makeBigTag(n int) *Tag {
	tag := NewTag(4)
	for i := 0; i < n; i++ {
		tag.SetText(fmt.Sprintf("T%03d", i), fmt.Sprintf("value-%d", i))
	}
	return tag
}

func BenchmarkFindHot(b *testing.B) {
	tag := makeBigTag(40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tag.Find("T039")
	}
}

func BenchmarkTextHot(b *testing.B) {
	tag := makeBigTag(40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tag.Text("T039")
	}
}

func BenchmarkFindAllHot(b *testing.B) {
	tag := makeBigTag(40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tag.FindAll("T039")
	}
}
