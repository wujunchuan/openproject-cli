package printer

import "testing"

func TestWithSpinnerSilent(t *testing.T) {
	SetSilent(true)
	defer SetSilent(false)

	called := false
	result, err := WithSpinner(func() (int, error) {
		called = true
		return 42, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
	if !called {
		t.Fatal("function was not called")
	}
}
