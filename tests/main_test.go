package tests

import (
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

const (
	defaultTestMemoryLimit = 768 << 20
	defaultTestGCPercent   = 25
)

var testCollectionEvery = 8

func TestMain(m *testing.M) {
	configureTestRuntime()
	code := m.Run()
	collectTestHeap()
	os.Exit(code)
}

func configureTestRuntime() {
	if limit, ok := configuredMemoryLimit(); ok {
		debug.SetMemoryLimit(limit)
	}
	if percent, ok := configuredGCPercent(); ok {
		debug.SetGCPercent(percent)
	}
	if every, ok := configuredCollectionEvery(); ok {
		testCollectionEvery = every
	}
	collectTestHeap()
}

func configuredMemoryLimit() (int64, bool) {
	if raw, ok := os.LookupEnv("MTAG_TEST_MEMORY_LIMIT"); ok {
		if n, err := parseByteSize(raw); err == nil && n > 0 {
			return n, true
		}
	}
	if _, ok := os.LookupEnv("GOMEMLIMIT"); ok {
		return 0, false
	}
	return defaultTestMemoryLimit, true
}

func configuredGCPercent() (int, bool) {
	if raw, ok := os.LookupEnv("MTAG_TEST_GC_PERCENT"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n >= 0 {
			return n, true
		}
	}
	if _, ok := os.LookupEnv("GOGC"); ok {
		return 0, false
	}
	return defaultTestGCPercent, true
}

func configuredCollectionEvery() (int, bool) {
	if raw, ok := os.LookupEnv("MTAG_TEST_GC_EVERY"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
			return n, true
		}
	}
	return testCollectionEvery, true
}

func maybeCollectEvery(i, every int) {
	if every <= 0 || (i+1)%every != 0 {
		return
	}
	collectTestHeap()
}

func collectTestHeap() {
	runtime.GC()
	debug.FreeOSMemory()
}

func parseByteSize(raw string) (int64, error) {
	s := strings.TrimSpace(strings.ToUpper(raw))
	s = strings.ReplaceAll(s, "IB", "I")
	s = strings.TrimSuffix(s, "B")
	multiplier := int64(1)
	for _, suffix := range []struct {
		name string
		mult int64
	}{
		{"KI", 1 << 10},
		{"MI", 1 << 20},
		{"GI", 1 << 30},
		{"TI", 1 << 40},
		{"K", 1 << 10},
		{"M", 1 << 20},
		{"G", 1 << 30},
		{"T", 1 << 40},
	} {
		if strings.HasSuffix(s, suffix.name) {
			s = strings.TrimSpace(strings.TrimSuffix(s, suffix.name))
			multiplier = suffix.mult
			break
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n * multiplier, nil
}
