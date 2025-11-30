package service

import (
	"testing"

	"github.com/forbearing/gst/types/consts"
)

func BenchmarkServiceKey(b *testing.B) {
	for b.Loop() {
		_ = serviceKey[*testUser, *testUser, *testUser](consts.PHASE_CREATE)
	}
}
