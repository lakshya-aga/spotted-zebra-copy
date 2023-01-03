package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetValues(t *testing.T) {
	store := NewStore(testDB)

	n := 5
	errs := make(chan error)
	results := make(chan GetValuesResult)

	// run n concurrent queries
	for i := 0; i < n; i++ {
		go func() {
			result, err := store.GetValues(context.Background())
			errs <- err
			results <- result
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
		result := <-results
		require.NotEmpty(t, result)

		// check param
		param := result.Params
		require.NotEmpty(t, param)
		paramDate, err := store.GetLatestParamDate(context.Background())
		require.NoError(t, err)
		for _, p := range param {
			require.NotEmpty(t, p)
			require.Equal(t, p.Date, paramDate)
		}

		// check stats
		stats := result.Stats
		require.NotEmpty(t, stats)
		statsDate, err := store.GetLatestStatsDate(context.Background())
		require.NoError(t, err)
		for _, p := range stats {
			require.NotEmpty(t, p)
			require.Equal(t, p.Date, statsDate)
		}

		// check corr
		corr := result.Corrpair
		require.NotEmpty(t, corr)
		corrDate, err := store.GetLatestCorrDate(context.Background())
		require.NoError(t, err)
		for _, p := range stats {
			require.NotEmpty(t, p)
			require.Equal(t, p.Date, corrDate)
		}
	}
}
