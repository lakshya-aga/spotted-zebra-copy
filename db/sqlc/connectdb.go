package db

import (
	"context"
)

type GetValuesResult struct {
	Params      []Modelparameter
	Stats       []Statistic
	Corrpair    []Corrpair
	LatestPrice []GetLatestPriceRow
}

type GetBacktestValuesResult struct {
	Params   []Modelparameter
	Stats    []Statistic
	Corrpair []Corrpair
	Date     []string
}

func (store *SQLStore) GetValues(ctx context.Context) (GetValuesResult, error) {
	var result GetValuesResult
	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		paramDate, err := q.GetLatestParamDate(ctx)
		if err != nil {
			return err
		}
		result.Params, err = q.GetParam(ctx, paramDate)
		if err != nil {
			return err
		}

		statsDate, err := q.GetLatestStatsDate(ctx)
		if err != nil {
			return err
		}
		result.Stats, err = q.GetStats(ctx, statsDate)
		if err != nil {
			return err
		}

		corrDate, err := q.GetLatestCorrDate(ctx)
		if err != nil {
			return err
		}
		result.Corrpair, err = q.GetCorr(ctx, corrDate)
		if err != nil {
			return err
		}

		result.LatestPrice, err = q.GetLatestPrice(ctx)
		if err != nil {
			return err
		}

		return err
	})
	return result, err
}

func (store *SQLStore) GetBacktestValues(ctx context.Context) (GetBacktestValuesResult, error) {
	var result GetBacktestValuesResult
	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Params, err = q.GetAllParam(ctx)
		if err != nil {
			return err
		}

		result.Stats, err = q.GetAllStats(ctx)
		if err != nil {
			return err
		}

		result.Corrpair, err = q.GetAllCorr(ctx)
		if err != nil {
			return err
		}

		result.Date, err = q.GetAllDate(ctx)
		if err != nil {
			return err
		}

		return err
	})
	return result, err
}
