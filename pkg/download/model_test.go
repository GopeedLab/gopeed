package download

import (
	"testing"
	"time"
)

func TestTransferSpeedWithUnevenSamples(t *testing.T) {
	var task Task
	for _, sample := range []struct {
		bytes   int64
		seconds float64
	}{
		{1000, 1},
		{2000, 2},
		{4000, 4},
		{6000, 6},
	} {
		if got := task.updateSpeed(sample.bytes, sample.seconds); got != 1000 {
			t.Errorf("download speed after %v seconds = %d, want 1000", sample.seconds, got)
		}
		if got := task.updateUploadSpeed(sample.bytes, sample.seconds); got != 1000 {
			t.Errorf("upload speed after %v seconds = %d, want 1000", sample.seconds, got)
		}
	}
}

func TestCalcSpeedResetOnRollback(t *testing.T) {
	speedArr := []speedSample{{1024, 1}, {2048, 1}, {4096, 1}}

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

func TestSampleSecondsUsesThePreviousSuccessfulSample(t *testing.T) {
	start := time.Now()
	var last time.Time
	if got := sampleSeconds(&last, start); got != 0 {
		t.Fatalf("first sample elapsed = %v, want 0", got)
	}
	if got := sampleSeconds(&last, start.Add(1500*time.Millisecond)); got != 1.5 {
		t.Fatalf("elapsed after a delayed sample = %v, want 1.5", got)
	}
	if got := sampleSeconds(&last, start.Add(2*time.Second)); got != 0.5 {
		t.Fatalf("elapsed after the next sample = %v, want 0.5", got)
	}
}
