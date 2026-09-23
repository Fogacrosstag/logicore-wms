package domain

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEligibilityAndRanking(t *testing.T) {
	base := Candidate{ID: "a", WarehouseID: "w", MaxWeight: 100, MaxVolume: 100, StorageType: "NORMAL", ZoneType: "STORAGE", Status: "AVAILABLE"}
	near, ok := Score(base, "w", "NORMAL", 10, 10)
	require.True(t, ok)
	far := base
	far.Distance = 100
	score, ok := Score(far, "w", "NORMAL", 10, 10)
	require.True(t, ok)
	require.Greater(t, near, score)
	for _, status := range []string{"BLOCKED", "MAINTENANCE"} {
		c := base
		c.Status = status
		_, ok = Score(c, "w", "NORMAL", 10, 10)
		require.False(t, ok)
	}
	_, ok = Score(base, "w", "COLD", 10, 10)
	require.False(t, ok)
	_, ok = Score(base, "other", "NORMAL", 10, 10)
	require.False(t, ok)
	_, ok = Score(base, "w", "NORMAL", 101, 10)
	require.False(t, ok)
	c := base
	c.CurrentVolume = 95
	_, ok = Score(c, "w", "NORMAL", 10, 10)
	require.False(t, ok)
}
