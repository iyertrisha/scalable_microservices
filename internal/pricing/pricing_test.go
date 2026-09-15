package pricing

import "testing"

func TestCalculate(t *testing.T) {
	amt, level := Calculate(10, "M", "normal", "normal")
	if level != "normal" || amt < 80 {
		t.Fatalf("unexpected quote amount=%d level=%s", amt, level)
	}
	high, hl := Calculate(10, "M", "normal", "high")
	if hl != "high" || high <= amt {
		t.Fatalf("high demand should cost more: base=%d high=%d", amt, high)
	}
	vh, vl := Calculate(10, "M", "normal", "very_high")
	if vl != "very_high" || vh <= high {
		t.Fatalf("very_high should cost more: high=%d vh=%d", high, vh)
	}
}
