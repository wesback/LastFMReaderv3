package progress

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

// TestNewRealProgressBar tests creating a new RealProgressBar with various options.
func TestNewRealProgressBar(t *testing.T) {
	tests := []struct {
		name        string
		total       int64
		description string
		opts        []Option
	}{
		{
			name:        "basic creation",
			total:       100,
			description: "Test progress",
			opts:        nil,
		},
		{
			name:        "with custom width",
			total:       50,
			description: "Custom width",
			opts:        []Option{WithWidth(80)},
		},
		{
			name:        "with ASCII style",
			total:       200,
			description: "ASCII style",
			opts:        []Option{WithStyle(StyleASCII)},
		},
		{
			name:        "disabled colors",
			total:       100,
			description: "No colors",
			opts:        []Option{WithColors(false)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := NewRealProgressBar(tt.total, tt.description, tt.opts...)
			if bar == nil {
				t.Fatal("Expected ProgressBar, got nil")
			}

			if bar.total != tt.total {
				t.Errorf("Expected total %d, got %d", tt.total, bar.total)
			}

			if bar.description != tt.description {
				t.Errorf("Expected description %q, got %q", tt.description, bar.description)
			}

			if bar.finished {
				t.Error("New bar should not be finished")
			}

			if bar.state != StateActive {
				t.Errorf("Expected StateActive, got %v", bar.state)
			}
		})
	}
}

// TestProgressBarLifecycle tests the basic lifecycle: Start → Add → Finish

// TestProgressBarLifecycle tests the basic lifecycle: Start → Add → Finish
func TestProgressBarLifecycle(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	// Start
	err := bar.Start(10, "Processing items")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if bar.IsFinished() {
		t.Error("Bar should not be finished after Start")
	}

	// Add increments
	for i := 0; i < 10; i++ {
		err = bar.Add(1)
		if err != nil {
			t.Fatalf("Add(%d) failed: %v", i, err)
		}
	}

	// Finish
	err = bar.Finish("Complete")
	if err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	if !bar.IsFinished() {
		t.Error("Bar should be finished after Finish")
	}

	if bar.state != StateSuccess {
		t.Errorf("Expected StateSuccess, got %v", bar.state)
	}
}

// TestProgressBarSetCurrent tests setting current value directly
func TestProgressBarSetCurrent(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(100, "Test")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	tests := []struct {
		name      string
		value     int64
		expectErr bool
	}{
		{"valid value", 50, false},
		{"zero value", 0, false},
		{"max value", 100, false},
		{"exceeds total", 101, true},
		{"negative value", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bar.SetCurrent(tt.value)
			if tt.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestProgressBarValidation tests validation of Add and SetCurrent
func TestProgressBarValidation(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(100, "Validation test")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Negative increment
	err = bar.Add(-5)
	if err == nil {
		t.Error("Expected error for negative increment")
	}

	// Exceed total
	err = bar.SetCurrent(50)
	if err != nil {
		t.Fatalf("SetCurrent(50) failed: %v", err)
	}

	err = bar.Add(60) // Would make current = 110
	if err == nil {
		t.Error("Expected error when exceeding total")
	}

	// Finish the bar
	err = bar.Finish("Done")
	if err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	// Operations on finished bar should fail
	err = bar.Add(1)
	if err == nil {
		t.Error("Expected error when adding to finished bar")
	}

	err = bar.SetCurrent(10)
	if err == nil {
		t.Error("Expected error when setting current on finished bar")
	}

	// SetDescription doesn't fail on finished bar (just updates internal state)
	err = bar.SetDescription("New desc")
	if err != nil {
		t.Errorf("SetDescription should not fail on finished bar: %v", err)
	}
}

// TestProgressBarStateTransitions tests state transitions
func TestProgressBarStateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		finishFunc    func(*ProgressBar) error
		expectedState ProgressState
	}{
		{
			name: "success state",
			finishFunc: func(pb *ProgressBar) error {
				return pb.Finish("Success")
			},
			expectedState: StateSuccess,
		},
		{
			name: "error state",
			finishFunc: func(pb *ProgressBar) error {
				return pb.FinishWithError("Error occurred")
			},
			expectedState: StateError,
		},
		{
			name: "warning state",
			finishFunc: func(pb *ProgressBar) error {
				return pb.FinishWithWarning("Warning occurred")
			},
			expectedState: StateWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			bar := NewRealProgressBar(0, "", WithWriter(buf))

			err := bar.Start(10, "State transition test")
			if err != nil {
				t.Fatalf("Start failed: %v", err)
			}

			if bar.state != StateActive {
				t.Errorf("Expected StateActive initially, got %v", bar.state)
			}

			err = tt.finishFunc(bar)
			if err != nil {
				t.Fatalf("Finish function failed: %v", err)
			}

			if bar.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, bar.state)
			}

			if !bar.IsFinished() {
				t.Error("Bar should be finished")
			}
		})
	}
}

// TestProgressBarSpinnerTransition tests switching between progress and spinner modes
func TestProgressBarSpinnerTransition(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(100, "Spinner test")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if bar.state != StateActive {
		t.Errorf("Expected StateActive, got %v", bar.state)
	}

	// Switch to spinner
	err = bar.SwitchToSpinner()
	if err != nil {
		t.Fatalf("SwitchToSpinner failed: %v", err)
	}

	if bar.state != StateSpinner {
		t.Errorf("Expected StateSpinner, got %v", bar.state)
	}

	// Resume progress
	err = bar.ResumeProgress()
	if err != nil {
		t.Fatalf("ResumeProgress failed: %v", err)
	}

	if bar.state != StateActive {
		t.Errorf("Expected StateActive after resume, got %v", bar.state)
	}

	// Can't resume if not in spinner mode
	err = bar.ResumeProgress()
	if err == nil {
		t.Error("Expected error when resuming from non-spinner state")
	}
}

// TestProgressBarSetDescription tests updating the description
func TestProgressBarSetDescription(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(10, "Initial description")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if bar.description != "Initial description" {
		t.Errorf("Expected 'Initial description', got %q", bar.description)
	}

	err = bar.SetDescription("Updated description")
	if err != nil {
		t.Fatalf("SetDescription failed: %v", err)
	}

	if bar.description != "Updated description" {
		t.Errorf("Expected 'Updated description', got %q", bar.description)
	}
}

// TestProgressBarThreadSafety tests concurrent Add operations
func TestProgressBarThreadSafety(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	total := int64(1000)
	err := bar.Start(total, "Thread safety test")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Launch 100 goroutines, each adding 10
	numGoroutines := 100
	incrementPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementPerGoroutine; j++ {
				bar.Add(1)
			}
		}()
	}

	wg.Wait()

	expectedCurrent := int64(numGoroutines * incrementPerGoroutine)
	if bar.current != expectedCurrent {
		t.Errorf("Expected current %d, got %d", expectedCurrent, bar.current)
	}
}

// TestProgressBarParallelOperations tests multiple operations in parallel
func TestProgressBarParallelOperations(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(1000, "Parallel operations")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	var wg sync.WaitGroup

	// Goroutine 1: Add increments
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			bar.Add(1)
			time.Sleep(time.Microsecond)
		}
	}()

	// Goroutine 2: Update description
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			bar.SetDescription("Update " + string(rune('A'+i)))
			time.Sleep(10 * time.Microsecond)
		}
	}()

	// Goroutine 3: SetCurrent
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := int64(0); i < 50; i++ {
			bar.SetCurrent(i * 10)
			time.Sleep(20 * time.Microsecond)
		}
	}()

	wg.Wait()

	// Should complete without panics or race conditions
	if bar.IsFinished() {
		t.Error("Bar should not be finished yet")
	}
}

// TestProgressBarClear tests clearing the progress bar
func TestProgressBarClear(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(10, "Clear test")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = bar.Add(5)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	err = bar.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Should be able to continue after clear
	err = bar.Add(3)
	if err != nil {
		t.Fatalf("Add after Clear failed: %v", err)
	}
}

// TestProgressBarRestart tests starting an already started bar
func TestProgressBarRestart(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(10, "First start")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Second start should reset
	err = bar.Start(20, "Second start")
	if err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	if bar.total != 20 {
		t.Errorf("Expected total 20 after restart, got %d", bar.total)
	}

	if bar.description != "Second start" {
		t.Errorf("Expected description 'Second start', got %q", bar.description)
	}

	if bar.current != 0 {
		t.Errorf("Expected current 0 after restart, got %d", bar.current)
	}
}

// TestProgressBarMultipleFinish tests calling finish multiple times
func TestProgressBarMultipleFinish(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewRealProgressBar(0, "", WithWriter(buf))

	err := bar.Start(10, "Multiple finish test")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = bar.Finish("First finish")
	if err != nil {
		t.Fatalf("First Finish failed: %v", err)
	}

	// Second finish should be a no-op, not an error
	err = bar.Finish("Second finish")
	if err != nil {
		t.Errorf("Second Finish should be no-op, got error: %v", err)
	}
}
