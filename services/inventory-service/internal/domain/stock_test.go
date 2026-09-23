package domain

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStockGuards(t *testing.T) {
	require.Equal(t, int64(85), Available(100, 15))
	for _, q := range []int64{-1, 0, 86} {
		require.Error(t, CheckQuantity(q, 85))
	}
	require.NoError(t, CheckQuantity(85, 85))
}
