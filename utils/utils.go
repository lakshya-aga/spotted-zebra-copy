package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Holidays struct {
	Exch   string `json:"exchange"`
	Date   string `json:"date"`
	Status string `json:"status"`
}

const Layout = "2006-01-02"

var NYSE = []string{"2022-01-01", "2022-01-17", "2022-02-21", "2022-04-15", "2022-05-30", "2022-06-20", "2022-07-04", "2022-09-05", "2022-11-24", "2022-12-26", "2023-01-02", "2023-01-16", "2023-02-20", "2023-04-07", "2023-05-29", "2023-06-19", "2023-07-04", "2023-09-04", "2023-11-23", "2023-12-25", "2024-01-01", "2024-01-15", "2024-02-19", "2024-07-04", "2024-09-02", "2024-11-28", "2024-12-25"}

func UpcomingHols(exch string) ([]time.Time, error) {
	var exch_hol []time.Time
	hols, err := get("https://api.polygon.io/v1/marketstatus/upcoming", []Holidays{})
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(hols); i++ {
		if hols[i].Exch == exch && hols[i].Status == "closed" {
			t, _ := time.Parse(Layout, hols[i].Date)
			exch_hol = append(exch_hol, t)
		}
	}
	return exch_hol, err
}

// Convert holidays from string to time.Time format
func Hols(s []string) ([]time.Time, error) {
	h := make([]time.Time, len(s))
	var err error
	var d time.Time
	for i, v := range s {
		d, err = time.Parse("2006-01-02", v)
		if err != nil {
			return nil, err
		}
		h[i] = d
	}
	return h, err
}

func IsHol(d time.Time, hols []time.Time) bool {
	if hols == nil {
		return false
	}
	for _, v := range hols {
		if d.Equal(v) {
			return true
		}
	}
	return false
}

func IsWeekday(d time.Time) bool {
	if d.Weekday() > 0 && d.Weekday() < 6 {
		return true
	}
	return false
}

func AdjustFollowing(d time.Time, hols []time.Time) time.Time {
	for {
		if IsHol(d, hols) || !IsWeekday(d) {
			d = d.AddDate(0, 0, 1)
		} else {
			return d
		}
	}
}

// Return a list of business days from (and including) a start date to (and including) an end date according to a holiday calendar
func ListBusinessDates(start time.Time, end time.Time, hols []time.Time) ([]time.Time, error) {
	if end.Before(start) {
		err := errors.New("end date must be later than start date")
		return nil, err
	}
	var out = []time.Time{start}
	for {
		start = AdjustFollowing(start.AddDate(0, 0, 1), hols)
		if start.After(end) {
			return out, nil
		}
		out = append(out, start)
	}
}

// Return a map of monte-carlo and knock-out barrier observation dates.
// Frequency (freq) and tenor arguments are in number of months.
func GenerateDates(start time.Time, tenor, freq int) (map[string][]time.Time, error) {
	out := make(map[string][]time.Time, 2)
	n := tenor / freq
	kodates := make([]time.Time, n)
	hols, err := Hols(NYSE)
	if err != nil {
		return nil, err
	}
	for i := 0; i < n; i++ {
		kodates[i] = AdjustFollowing(start.AddDate(0, (i+1)*freq, 0), hols)
	}
	mcdates, err := ListBusinessDates(start, kodates[len(kodates)-1], hols)
	if err != nil {
		return nil, err
	}
	out["mcdates"] = mcdates
	out["kodates"] = kodates
	return out, err
}

func get[DataType []Holidays](url string, target DataType) (result DataType, err error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", `Bearer 3X8wQrb0pH9gaNJvY__sq1UohDdHfVt3`)
	if err != nil {
		return DataType{}, nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DataType{}, nil
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	result = target
	return result, nil
}

// Minimum of a slice
func MinSlice(a []float64) float64 {
	var m float64
	for i, e := range a {
		if i == 0 || e < m {
			m = e
		}
	}
	return m
}

func IsIn(t time.Time, ts []time.Time) bool {
	for _, v := range ts {
		if t.Equal(v) {
			return true
		}
	}
	return false
}

func format(stocks []string) []string {
	if len(stocks) == 0 {
		return []string{}
	}
	sort.Strings(stocks)
	for s := 0; s < len(stocks); s++ {
		stocks[s] = strings.ToUpper(stocks[s])
	}
	var unique []string

	for _, v := range stocks {
		skip := false
		for _, u := range unique {
			if v == u {
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, v)
		}
	}
	return unique
}

func Filter(selectStocks, DefaultStocks []string) ([]string, map[string]int, error) {
	selectStocks = format(selectStocks)
	stockIndex := map[string]int{}
	var stocks []string
	for i, v := range DefaultStocks {
		for j := range selectStocks {
			if v == selectStocks[j] {
				stockIndex[v] = i
				stocks = append(stocks, v)
			}
		}
	}
	if len(stocks) == 0 {
		err := errors.New("there is no available stocks")
		return nil, nil, err
	}
	return stocks, stockIndex, nil
}
