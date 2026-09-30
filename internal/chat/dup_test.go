package chat

import "testing"

// 覆盖 dupTracker：连续同参、窗口内非连续同参、窗口挤出后计数重置。

func TestDupTrackerConsecutive(t *testing.T) {
	d := newDupTracker(3, 12)
	if d.add("a") || d.add("a") {
		t.Fatal("should not trip before 3")
	}
	if !d.add("a") {
		t.Fatal("should trip on 3rd consecutive")
	}
	if d.count("a") != 3 {
		t.Fatalf("count=%d", d.count("a"))
	}
}

func TestDupTrackerNonConsecutive(t *testing.T) {
	d := newDupTracker(3, 12)
	// A B A B A → 第 3 次 A 熔断
	seq := []string{"a", "b", "a", "b", "a"}
	for i, k := range seq {
		hit := d.add(k)
		if i < 4 && hit {
			t.Fatalf("unexpected trip at i=%d key=%s", i, k)
		}
		if i == 4 && !hit {
			t.Fatal("expected trip on 3rd a")
		}
	}
}

func TestDupTrackerWindowEvicts(t *testing.T) {
	d := newDupTracker(2, 3)
	_ = d.add("a")
	_ = d.add("b")
	_ = d.add("c") // window: b,c — a 被挤出
	if d.add("a") {
		t.Fatal("first a after eviction should not trip (count=1)")
	}
	if !d.add("a") {
		t.Fatal("second a in window should trip")
	}
}
