package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScheduleToCronExpr(t *testing.T) {
	type testCase struct {
		schedule     map[string]interface{}
		expectedCron string
	}

	cases := []testCase{
		{
			schedule:     map[string]interface{}{"minute": 0},
			expectedCron: " * * * * *",
		},
		{
			schedule:     map[string]interface{}{"minute": "0", "hour": "0", "dom": "1"},
			expectedCron: "0 0 1 * *",
		},
	}

	for _, tc := range cases {
		got := scheduleToCronExpr(tc.schedule)
		assert.Equal(t, tc.expectedCron, got)
	}
}
