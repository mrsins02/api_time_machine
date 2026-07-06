package diff_test

import (
	"testing"

	"github.com/api-time-machine/api_time_machine/internal/diff"
)

func TestDiffChangedFields(t *testing.T) {
	engine := diff.New()
	result := engine.Compare(
		map[string]any{"name": "Ali", "email": "old@example.com"},
		map[string]any{"name": "Ali", "email": "new@example.com"},
	)
	if len(result.Changed) != 1 || result.Changed[0] != "email" {
		t.Fatalf("expected email changed, got %v", result.Changed)
	}
}
