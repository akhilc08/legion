package stdout

import (
	"testing"
)

var sink bool
var sinkControl interface{}

func BenchmarkIsControlLine_Hit(b *testing.B) {
	line := `CONDUCTOR_HIRE {"role_title":"engineer","system_prompt":"you are an engineer","budget_allocation":50000}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink = IsControlLine(line)
	}
}

func BenchmarkIsControlLine_Miss(b *testing.B) {
	line := "2024/01/01 00:00:00 INFO agent started and ready to receive tasks"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink = IsControlLine(line)
	}
}

func BenchmarkParseLine_Hire(b *testing.B) {
	line := `CONDUCTOR_HIRE {"role_title":"engineer","system_prompt":"you are an engineer","budget_allocation":50000}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c, _ := ParseLine(line)
		sinkControl = c
	}
}

func BenchmarkParseLine_Heartbeat(b *testing.B) {
	line := `CONDUCTOR_HEARTBEAT {"ts":1711929600}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c, _ := ParseLine(line)
		sinkControl = c
	}
}

func BenchmarkDecodeHire(b *testing.B) {
	line := `CONDUCTOR_HIRE {"role_title":"engineer","system_prompt":"you are an engineer","budget_allocation":50000}`
	parsed, _ := ParseLine(line)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h, _ := DecodeHire(parsed.Payload)
		sinkControl = h
	}
}
