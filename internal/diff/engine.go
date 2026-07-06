package diff

import (
	"reflect"

	"github.com/api-time-machine/api_time_machine/internal/domain"
)

type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Compare(from, to map[string]any) domain.DiffResult {
	result := domain.DiffResult{
		From:   from,
		To:     to,
		Fields: map[string]domain.FieldChange{},
	}

	seen := map[string]struct{}{}
	for k, toVal := range to {
		seen[k] = struct{}{}
		fromVal, ok := from[k]
		if !ok {
			result.Changed = append(result.Changed, k)
			result.Fields[k] = domain.FieldChange{From: nil, To: toVal}
			continue
		}
		if !reflect.DeepEqual(fromVal, toVal) {
			result.Changed = append(result.Changed, k)
			result.Fields[k] = domain.FieldChange{From: fromVal, To: toVal}
		}
	}

	for k, fromVal := range from {
		if _, ok := seen[k]; ok {
			continue
		}
		result.Changed = append(result.Changed, k)
		result.Fields[k] = domain.FieldChange{From: fromVal, To: nil}
	}

	return result
}
