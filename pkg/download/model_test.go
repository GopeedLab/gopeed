package download

import "testing"

func TestCalcSpeedResetOnRollback(t *testing.T) {
	speedArr := []int64{1024, 2048, 4096}

	if got := calcSpeed(&speedArr, -512, 1); got != 0 {
		t.Fatalf("calcSpeed() = %d, want 0 after rollback", got)
	}
	if len(speedArr) != 0 {
		t.Fatalf("speed window len = %d, want 0 after rollback", len(speedArr))
	}

	if got := calcSpeed(&speedArr, 1024, 1); got != 1024 {
		t.Fatalf("calcSpeed() = %d, want 1024 after reset", got)
	}
}
