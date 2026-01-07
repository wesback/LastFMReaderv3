package progress

import "testing"

func TestNoOpProgressBar(t *testing.T) {
	bar := NewNoOpProgressBar()

	// Test all methods can be called without errors
	tests := []struct {
		name string
		fn   func() error
	}{
		{"Start", func() error { return bar.Start(100, "test") }},
		{"Add", func() error { return bar.Add(1) }},
		{"SetCurrent", func() error { return bar.SetCurrent(50) }},
		{"SetDescription", func() error { return bar.SetDescription("new desc") }},
		{"Finish", func() error { return bar.Finish("done") }},
		{"FinishWithError", func() error { return bar.FinishWithError("error") }},
		{"FinishWithWarning", func() error { return bar.FinishWithWarning("warning") }},
		{"SwitchToSpinner", func() error { return bar.SwitchToSpinner() }},
		{"ResumeProgress", func() error { return bar.ResumeProgress() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Errorf("%s() returned error: %v", tt.name, err)
			}
		})
	}

	// Test IsFinished
	if bar.IsFinished() {
		t.Error("NoOpProgressBar.IsFinished() should always return false")
	}
}

func TestNoOpProgressBarMultipleCalls(t *testing.T) {
	bar := NewNoOpProgressBar()

	// Test multiple calls don't cause issues
	for i := 0; i < 100; i++ {
		if err := bar.Add(1); err != nil {
			t.Errorf("Add() call %d returned error: %v", i, err)
		}
	}

	// Start multiple times should be fine
	if err := bar.Start(100, "first"); err != nil {
		t.Errorf("First Start() returned error: %v", err)
	}
	if err := bar.Start(200, "second"); err != nil {
		t.Errorf("Second Start() returned error: %v", err)
	}

	// Finish multiple times should be fine
	if err := bar.Finish("done 1"); err != nil {
		t.Errorf("First Finish() returned error: %v", err)
	}
	if err := bar.Finish("done 2"); err != nil {
		t.Errorf("Second Finish() returned error: %v", err)
	}
}
