package temporal

import (
	"strconv"
	"strings"
	"time"

	"github.com/api-time-machine/api_time_machine/internal/domain"
)

// ParseReplayQuery builds a replay query from HTTP-style parameters.
func ParseReplayQuery(at string, version string) (domain.ReplayQuery, error) {
	q := domain.ReplayQuery{}
	if at != "" {
		t, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return q, err
		}
		q.At = &t
	}
	if version != "" {
		n, err := strconv.ParseInt(version, 10, 64)
		if err != nil {
			return q, err
		}
		q.Version = &n
	}
	return q, nil
}

// ParseBound accepts RFC3339 timestamps or version numbers (optionally prefixed with "v").
func ParseBound(value string) (domain.ReplayQuery, error) {
	if value == "" {
		return domain.ReplayQuery{}, nil
	}
	if strings.HasPrefix(value, "v") {
		n, err := strconv.ParseInt(strings.TrimPrefix(value, "v"), 10, 64)
		if err != nil {
			return domain.ReplayQuery{}, err
		}
		return domain.ReplayQuery{Version: &n}, nil
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return domain.ReplayQuery{Version: &n}, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return domain.ReplayQuery{}, err
	}
	return domain.ReplayQuery{At: &t}, nil
}

func IsTemporal(q domain.ReplayQuery) bool {
	return domain.IsTemporal(q)
}
