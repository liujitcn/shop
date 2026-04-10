package event

import "testing"

func TestEventWeight(t *testing.T) {
	if got := EventWeight(EventTypePay); got != 10 {
		t.Fatalf("EventWeight(pay) = %v, want 10", got)
	}
	if got := EventWeight("unknown"); got != 0 {
		t.Fatalf("EventWeight(unknown) = %v, want 0", got)
	}
}

func TestAddBehaviorSummaryCount(t *testing.T) {
	got, err := AddBehaviorSummaryCount("", "click_count", 2)
	if err != nil {
		t.Fatalf("AddBehaviorSummaryCount returned error: %v", err)
	}
	if got != "{\"click_count\":2}" {
		t.Fatalf("AddBehaviorSummaryCount = %s", got)
	}
}
