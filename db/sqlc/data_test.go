package db

import (
	"context"
	"testing"
	"time"

	"github.com/banachtech/spotted-zebra/util"
	"github.com/stretchr/testify/require"
)

const Layout = "2006-01-02"

func createRandomParam(t *testing.T) []float64 {
	var params []float64
	for i := 0; i < 5; i++ {
		params = append(params, util.RandomFloats())
	}
	return params
}

func insertParams(t *testing.T) Modelparameter {
	params := createRandomParam(t)
	stock := util.RandomStock()
	date := time.Now().Format(Layout)
	arg := InsertParamParams{
		Date:   date,
		Ticker: stock,
		Sigma:  params[0],
		Alpha:  params[1],
		Beta:   params[2],
		Kappa:  params[3],
		Rho:    params[4],
	}
	result, err := testQueries.InsertParam(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, arg.Date, result.Date)
	require.Equal(t, arg.Ticker, result.Ticker)
	require.Equal(t, arg.Sigma, result.Sigma)
	require.Equal(t, arg.Alpha, result.Alpha)
	require.Equal(t, arg.Beta, result.Beta)
	require.Equal(t, arg.Kappa, result.Kappa)
	require.Equal(t, arg.Rho, result.Rho)
	return result
}

func TestInsertParam(t *testing.T) {
	insertParams(t)
}

func TestGetParam(t *testing.T) {
	param := insertParams(t)
	result, err := testQueries.GetParam(context.Background(), param.Date)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.NotEqual(t, len(result), 0)

	for _, p := range result {
		require.NotEmpty(t, p)
		require.Equal(t, p.Date, param.Date)
	}
}

func TestGetLatestParam(t *testing.T) {
	param := insertParams(t)
	result, err := testQueries.GetLatestParamDate(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, param.Date, result)
}

func insertCorrs(t *testing.T) Corrpair {
	corr := util.RandomFloats()
	stock1 := util.RandomStock()
	stock2 := util.RandomStock()
	date := time.Now().Format(Layout)
	arg := InsertCorrParams{
		Date: date,
		X0:   stock1,
		X1:   stock2,
		Corr: corr,
	}
	result, err := testQueries.InsertCorr(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, arg.Date, result.Date)
	require.Equal(t, arg.X0, result.X0)
	require.Equal(t, arg.X1, result.X1)
	require.Equal(t, arg.Corr, result.Corr)
	return result
}

func TestInsertCorr(t *testing.T) {
	insertCorrs(t)
}

func TestGetLatestCorr(t *testing.T) {
	param := insertCorrs(t)
	result, err := testQueries.GetLatestCorrDate(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, param.Date, result)
}

func TestGetCorr(t *testing.T) {
	param := insertCorrs(t)
	result, err := testQueries.GetCorr(context.Background(), param.Date)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	for _, p := range result {
		require.NotEmpty(t, p)
		require.Equal(t, p.Date, param.Date)
	}
}

func insertStats(t *testing.T) Statistic {
	mean := util.RandomFloats()
	fixing := util.RandomFloats()
	index := util.RandomInt(0, 9)
	stock := util.RandomStock()
	date := time.Now().Format(Layout)
	arg := InsertStatParams{
		Date:   date,
		Ticker: stock,
		Index:  index,
		Mean:   mean,
		Fixing: fixing,
	}
	result, err := testQueries.InsertStat(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, arg.Date, result.Date)
	require.Equal(t, arg.Ticker, result.Ticker)
	require.Equal(t, arg.Index, result.Index)
	require.Equal(t, arg.Mean, result.Mean)
	require.Equal(t, arg.Fixing, result.Fixing)
	return result
}

func TestInsertStat(t *testing.T) {
	insertStats(t)
}

func TestGetStat(t *testing.T) {
	param := insertStats(t)
	result, err := testQueries.GetStats(context.Background(), param.Date)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	for _, p := range result {
		require.NotEmpty(t, p)
		require.Equal(t, p.Date, param.Date)
	}
}

func TestGetLatestStats(t *testing.T) {
	param := insertStats(t)
	result, err := testQueries.GetLatestStatsDate(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, param.Date, result)
}

func TestGetLatestPrice(t *testing.T) {
	result, err := testQueries.GetLatestPrice(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, result)
}
