package data

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"main/mc"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
	"gonum.org/v1/gonum/optimize"
	"gonum.org/v1/gonum/stat/distuv"
)

// helper function to get the http request and store into struct from polygon.io
func getPolygon[DataType TickerDetails | Tickers | TickerAggs](url string, target DataType) (result DataType, err error) {
	err = godotenv.Load()
	if err != nil {
		return target, err
	}
	req, err := http.NewRequest("GET", url, nil)
	key := os.Getenv("POLYGON_API_KEY")
	req.Header.Add("Authorization", fmt.Sprintf(`Bearer %s`, key))
	if err != nil {
		return target, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return target, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	result = target
	return result, nil
}

// helper function to get the http request and store into struct from alphavantage
func getAlphavantage[DataType Overview | AlphaData](url string, target DataType) (result DataType, err error) {
	err = godotenv.Load()
	if err != nil {
		return target, err
	}
	req, err := http.NewRequest("GET", url, nil)
	q := req.URL.Query()
	key := os.Getenv("ALPHAVANTAGE_API_KEY")
	q.Add("apikey", key)
	req.URL.RawQuery = q.Encode()
	if err != nil {
		return target, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return target, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	result = target
	return result, nil
}

// black-scholes model
func bs(k, s, sigma, T, dy, r float64, option string) float64 {
	x := sigma * math.Sqrt(T)
	d1 := (math.Log(s/k) + 0.5*sigma*sigma*T) / x
	d2 := d1 - x

	N := distuv.Normal{Mu: 0.0, Sigma: 1.0}

	premium := s*math.Exp(-dy*T)*N.CDF(d1) - k*math.Exp(-r*T)*N.CDF(d2)
	if option == "p" {
		premium = -s*math.Exp(-dy*T)*N.CDF(-d1) + k*math.Exp(-r*T)*N.CDF(-d2)
	}
	return premium
}

// price loss function
func loss(par []float64, p, k, s, T, dy, r float64, option string) float64 {
	par[0] = math.Exp(par[0])
	loss := math.Pow(p-bs(k, s, par[0], T, dy, r, option), 2)
	return loss
}

// fitting the blackscholes model
func fit(p, k, s, T, dy, r float64, option string) float64 {
	par := []float64{math.Log(0.5)}
	problem := optimize.Problem{
		Func: func(par []float64) float64 {
			return loss(par, p, k, s, T, dy, r, option)
		},
	}
	res, err := optimize.Minimize(problem, par, nil, &optimize.NelderMead{})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	result := math.Exp(res.X[0])
	return result
}

// helper function to create json file
func createJson[T map[string][]Data | map[string][]string | map[string]mc.Model | Stat](raw T, filename string) error {
	data, err := json.MarshalIndent(raw, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// helper function to open tickers.json
func Open[T Model | []string | Stat](filename string, target T) (T, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return target, err
	}
	err = json.Unmarshal([]byte(file), &target)
	if err != nil {
		return target, err
	}
	return target, nil
}

// progress bar initialization
func progressBar(length int) *progressbar.ProgressBar {
	bar := progressbar.NewOptions(
		length,
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(20),
		progressbar.OptionSetVisibility(true),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	return bar
}

// minimum length
func minLength(data [][]float64) int {
	min := len(data[0])
	if len(data) == 1 {
		return min
	}
	for i := 1; i < len(data); i++ {
		if len(data[i]) < min {
			min = len(data[i])
		}
	}
	return min
}

// helper function to open csv file
func loadMktData(data []Data) ([][]float64, error) {
	var x [][]float64
	for i := 0; i < len(data); i++ {
		x = append(x, []float64{data[i].K, data[i].T, data[i].Ivol})
	}
	return x, nil
}

// get the index of the sorted stocks
func stockIndex(stocks []string) map[string]int {
	result := map[string]int{}
	for k, v := range stocks {
		result[v] = k
	}
	return result
}

// check if update is required
func UpdateRequired(target string, db *sql.DB, date string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT DISTINCT "Date" FROM "%s" ORDER BY "Date" DESC`, target))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var dates []string
	for rows.Next() {
		var dt string
		err = rows.Scan(&dt)
		if err != nil {
			return false, err
		}
		dates = append(dates, dt)
	}
	if len(dates) == 0 {
		return true, nil
	}
	latest, _ := time.Parse(Layout, dates[0])
	today, _ := time.Parse(Layout, date)
	if latest.Equal(today) {
		return false, nil
	} else {
		return true, nil
	}
}
