package group

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	require.Nil(t, nil)
	tt := []struct {
		name       string
		groupCalls [][]string
		expected   [][]string
	}{
		{
			name:       "happy path",
			groupCalls: [][]string{{"1", "2", "3"}},
			expected:   [][]string{{"1", "2", "3"}},
		},
		{
			name:       "skip if not change",
			groupCalls: [][]string{{"1", "2", "3"}, {"1", "2", "3"}, {"1", "2", "3"}},
			expected:   [][]string{{"1", "2", "3"}},
		},
		{
			name:       "multichange",
			groupCalls: [][]string{{"1", "2", "3"}, {"2", "3"}, {"1", "2", "3"}},
			expected:   [][]string{{"1", "2", "3"}, {"2", "3"}, {"1", "2", "3"}},
		},
		{
			name:       "sort preserved",
			groupCalls: [][]string{{"3", "2", "0"}, {"0", "2", "3"}, {"0", "2", "3"}},
			expected:   [][]string{{"3", "2", "0"}},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := make(chan time.Time)

			ctx, cancel := context.WithCancel(context.Background())

			var notifications [][]string

			notifyFn := func(ips []string) error {
				notifications = append(notifications, ips)
				return nil
			}
			groupFn := func(_ context.Context) ([]string, error) {
				var result []string
				result, tc.groupCalls = tc.groupCalls[0], tc.groupCalls[1:]
				return result, nil

			}

			go watch(ctx, notifyFn, groupFn, c)

			ticks := len(tc.groupCalls)

			for i := 0; i < ticks; i++ {
				c <- time.Now()
			}

			cancel()

			require.Len(t, tc.expected, len(notifications))

		})
	}
}
