package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/banachtech/spotted-zebra/db/mock"
	db "github.com/banachtech/spotted-zebra/db/sqlc"
	"github.com/banachtech/spotted-zebra/mc"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/mat"
)

func TestBackTest(t *testing.T) {
	values := db.GetBacktestValuesResult{
		Params: []db.Modelparameter{
			{Date: "2022-12-28", Ticker: "AAPL", Sigma: 0.38956884573910466, Alpha: 0.31754204762725213, Beta: 0.09668058826922904, Kappa: 18.55196217354717, Rho: -0.08156231110497626},
			{Date: "2022-12-28", Ticker: "AMZN", Sigma: 0.46575245263553094, Alpha: 0.2881529498901402, Beta: 0.2509865800514912, Kappa: 26.79220159521858, Rho: 0.24565297426344432},
			{Date: "2022-12-28", Ticker: "AVGO", Sigma: 0.33169818989315536, Alpha: 0.414590139046433, Beta: 0.4096664715295601, Kappa: 31.15386811469867, Rho: -0.2467638237838846},
			{Date: "2022-12-28", Ticker: "GOOG", Sigma: 0.38530278754601816, Alpha: 0.32351244056850875, Beta: 0.23109740907748258, Kappa: 48.112953769723696, Rho: 0.08997289925658783},
			{Date: "2022-12-28", Ticker: "INTC", Sigma: 0.44902724945877087, Alpha: 0.3393147453878549, Beta: 0.09350071076854136, Kappa: 41.58003923239355, Rho: 0.14048176577862234},
			{Date: "2022-12-28", Ticker: "META", Sigma: 0.5579224130648748, Alpha: 0.33334852472999166, Beta: 0.7514902119406026, Kappa: 17.90556500752665, Rho: -0.24208788705767267},
			{Date: "2022-12-28", Ticker: "MSFT", Sigma: 0.3432738277580456, Alpha: 0.3167856091036935, Beta: 0.09730997514366278, Kappa: 50.74003730490386, Rho: -0.06808543258266977},
			{Date: "2022-12-28", Ticker: "NVDA", Sigma: 0.6116133868757551, Alpha: 0.2823462216468162, Beta: 0.7126919633365246, Kappa: 23.622843456182757, Rho: -0.19590098109355483},
			{Date: "2022-12-28", Ticker: "QCOM", Sigma: 0.44874083366181716, Alpha: 0.29891068775458296, Beta: 0.2740408581474579, Kappa: 31.811198308053157, Rho: -0.00271927585342504},
			{Date: "2022-12-28", Ticker: "TSLA", Sigma: 0.926280232995074, Alpha: 0.09316279525141707, Beta: 0.11993430192118938, Kappa: 167.74229696983923, Rho: 0.9999999982622454},

			{Date: "2022-12-27", Ticker: "AAPL", Sigma: 0.38956884573910466, Alpha: 0.31754204762725213, Beta: 0.09668058826922904, Kappa: 18.55196217354717, Rho: -0.08156231110497626},
			{Date: "2022-12-27", Ticker: "AMZN", Sigma: 0.46575245263553094, Alpha: 0.2881529498901402, Beta: 0.2509865800514912, Kappa: 26.79220159521858, Rho: 0.24565297426344432},
			{Date: "2022-12-27", Ticker: "AVGO", Sigma: 0.33169818989315536, Alpha: 0.414590139046433, Beta: 0.4096664715295601, Kappa: 31.15386811469867, Rho: -0.2467638237838846},
			{Date: "2022-12-27", Ticker: "GOOG", Sigma: 0.38530278754601816, Alpha: 0.32351244056850875, Beta: 0.23109740907748258, Kappa: 48.112953769723696, Rho: 0.08997289925658783},
			{Date: "2022-12-27", Ticker: "INTC", Sigma: 0.44902724945877087, Alpha: 0.3393147453878549, Beta: 0.09350071076854136, Kappa: 41.58003923239355, Rho: 0.14048176577862234},
			{Date: "2022-12-27", Ticker: "META", Sigma: 0.5579224130648748, Alpha: 0.33334852472999166, Beta: 0.7514902119406026, Kappa: 17.90556500752665, Rho: -0.24208788705767267},
			{Date: "2022-12-27", Ticker: "MSFT", Sigma: 0.3432738277580456, Alpha: 0.3167856091036935, Beta: 0.09730997514366278, Kappa: 50.74003730490386, Rho: -0.06808543258266977},
			{Date: "2022-12-27", Ticker: "NVDA", Sigma: 0.6116133868757551, Alpha: 0.2823462216468162, Beta: 0.7126919633365246, Kappa: 23.622843456182757, Rho: -0.19590098109355483},
			{Date: "2022-12-27", Ticker: "QCOM", Sigma: 0.44874083366181716, Alpha: 0.29891068775458296, Beta: 0.2740408581474579, Kappa: 31.811198308053157, Rho: -0.00271927585342504},
			{Date: "2022-12-27", Ticker: "TSLA", Sigma: 0.926280232995074, Alpha: 0.09316279525141707, Beta: 0.11993430192118938, Kappa: 167.74229696983923, Rho: 0.9999999982622454},
		},
		Stats: []db.Statistic{
			{Date: "2022-12-28", Ticker: "AAPL", Mean: -0.0024065238291240444, Fixing: 130.03},
			{Date: "2022-12-28", Ticker: "AMZN", Mean: -0.005048579145832095, Fixing: 83.04},
			{Date: "2022-12-28", Ticker: "AVGO", Mean: 0.0029074417269861117, Fixing: 553.54},
			{Date: "2022-12-28", Ticker: "GOOG", Mean: -0.0017356250981795695, Fixing: 87.93},
			{Date: "2022-12-28", Ticker: "INTC", Mean: -0.0003667948755574481, Fixing: 25.94},
			{Date: "2022-12-28", Ticker: "META", Mean: -0.0022170263594699234, Fixing: 116.88},
			{Date: "2022-12-28", Ticker: "MSFT", Mean: 8.147414840341893e-05, Fixing: 236.96},
			{Date: "2022-12-28", Ticker: "NVDA", Mean: 0.0020500805926082105, Fixing: 141.21},
			{Date: "2022-12-28", Ticker: "QCOM", Mean: -0.0014103185849990312, Fixing: 109.46},
			{Date: "2022-12-28", Ticker: "TSLA", Mean: -0.015126507431615293, Fixing: 109.1},

			{Date: "2022-12-27", Ticker: "AAPL", Mean: -0.0024065238291240444, Fixing: 132.03},
			{Date: "2022-12-27", Ticker: "AMZN", Mean: -0.005048579145832095, Fixing: 82.04},
			{Date: "2022-12-27", Ticker: "AVGO", Mean: 0.0029074417269861117, Fixing: 555.54},
			{Date: "2022-12-27", Ticker: "GOOG", Mean: -0.0017356250981795695, Fixing: 84.93},
			{Date: "2022-12-27", Ticker: "INTC", Mean: -0.0003667948755574481, Fixing: 21.94},
			{Date: "2022-12-27", Ticker: "META", Mean: -0.0022170263594699234, Fixing: 113.88},
			{Date: "2022-12-27", Ticker: "MSFT", Mean: 8.147414840341893e-05, Fixing: 230.96},
			{Date: "2022-12-27", Ticker: "NVDA", Mean: 0.0020500805926082105, Fixing: 101.21},
			{Date: "2022-12-27", Ticker: "QCOM", Mean: -0.0014103185849990312, Fixing: 103.46},
			{Date: "2022-12-27", Ticker: "TSLA", Mean: -0.015126507431615293, Fixing: 106.1},
		},
		Corrpair: []db.Corrpair{
			{Date: "2022-12-28", X0: "AAPL", X1: "AMZN", Corr: 0.5616038538409025},
			{Date: "2022-12-28", X0: "AAPL", X1: "AVGO", Corr: 0.5135700399870929},
			{Date: "2022-12-28", X0: "AAPL", X1: "GOOG", Corr: 0.19453030218238854},
			{Date: "2022-12-28", X0: "AAPL", X1: "INTC", Corr: 0.4403990244849937},
			{Date: "2022-12-28", X0: "AAPL", X1: "META", Corr: 0.39386893288119945},
			{Date: "2022-12-28", X0: "AAPL", X1: "MSFT", Corr: 0.4817176327224049},
			{Date: "2022-12-28", X0: "AAPL", X1: "NVDA", Corr: 0.5023715611927752},
			{Date: "2022-12-28", X0: "AAPL", X1: "QCOM", Corr: 0.5238656511138837},
			{Date: "2022-12-28", X0: "AAPL", X1: "TSLA", Corr: 0.5498852123024683},
			{Date: "2022-12-28", X0: "AMZN", X1: "AVGO", Corr: 0.7952426819328237},
			{Date: "2022-12-28", X0: "AMZN", X1: "GOOG", Corr: 0.35188586495461893},
			{Date: "2022-12-28", X0: "AMZN", X1: "INTC", Corr: 0.697882967539835},
			{Date: "2022-12-28", X0: "AMZN", X1: "META", Corr: 0.7055127904435734},
			{Date: "2022-12-28", X0: "AMZN", X1: "MSFT", Corr: 0.7019173408166158},
			{Date: "2022-12-28", X0: "AMZN", X1: "NVDA", Corr: 0.7260722394406184},
			{Date: "2022-12-28", X0: "AMZN", X1: "QCOM", Corr: 0.7272532605981774},
			{Date: "2022-12-28", X0: "AMZN", X1: "TSLA", Corr: 0.9104063921158937},
			{Date: "2022-12-28", X0: "AVGO", X1: "GOOG", Corr: 0.47587714174716406},
			{Date: "2022-12-28", X0: "AVGO", X1: "INTC", Corr: 0.8082912216745181},
			{Date: "2022-12-28", X0: "AVGO", X1: "META", Corr: 0.7746634797304436},
			{Date: "2022-12-28", X0: "AVGO", X1: "MSFT", Corr: 0.8602847216586795},
			{Date: "2022-12-28", X0: "AVGO", X1: "NVDA", Corr: 0.6377690437133635},
			{Date: "2022-12-28", X0: "AVGO", X1: "QCOM", Corr: 0.8515255277410169},
			{Date: "2022-12-28", X0: "AVGO", X1: "TSLA", Corr: 0.8289691320432666},
			{Date: "2022-12-28", X0: "GOOG", X1: "INTC", Corr: 0.44132420140211615},
			{Date: "2022-12-28", X0: "GOOG", X1: "META", Corr: 0.6184997480729949},
			{Date: "2022-12-28", X0: "GOOG", X1: "MSFT", Corr: 0.4135973438859086},
			{Date: "2022-12-28", X0: "GOOG", X1: "NVDA", Corr: 0.45615499528776976},
			{Date: "2022-12-28", X0: "GOOG", X1: "QCOM", Corr: 0.40408105956893564},
			{Date: "2022-12-28", X0: "GOOG", X1: "TSLA", Corr: 0.3646675474220342},
			{Date: "2022-12-28", X0: "INTC", X1: "META", Corr: 0.798152957264765},
			{Date: "2022-12-28", X0: "INTC", X1: "MSFT", Corr: 0.8165799823040059},
			{Date: "2022-12-28", X0: "INTC", X1: "NVDA", Corr: 0.6697022210513814},
			{Date: "2022-12-28", X0: "INTC", X1: "QCOM", Corr: 0.8765020479402509},
			{Date: "2022-12-28", X0: "INTC", X1: "TSLA", Corr: 0.7980828372890855},
			{Date: "2022-12-28", X0: "META", X1: "MSFT", Corr: 0.7756588319810561},
			{Date: "2022-12-28", X0: "META", X1: "NVDA", Corr: 0.713536049308839},
			{Date: "2022-12-28", X0: "META", X1: "QCOM", Corr: 0.7643043773538588},
			{Date: "2022-12-28", X0: "META", X1: "TSLA", Corr: 0.7926189782086763},
			{Date: "2022-12-28", X0: "MSFT", X1: "NVDA", Corr: 0.5045881485309515},
			{Date: "2022-12-28", X0: "MSFT", X1: "QCOM", Corr: 0.7834928000403768},
			{Date: "2022-12-28", X0: "MSFT", X1: "TSLA", Corr: 0.7688564829562652},
			{Date: "2022-12-28", X0: "NVDA", X1: "QCOM", Corr: 0.6256210476226092},
			{Date: "2022-12-28", X0: "NVDA", X1: "TSLA", Corr: 0.7394663446100521},
			{Date: "2022-12-28", X0: "QCOM", X1: "TSLA", Corr: 0.7849776694898971},

			{Date: "2022-12-27", X0: "AAPL", X1: "AMZN", Corr: 0.5616038538409025},
			{Date: "2022-12-27", X0: "AAPL", X1: "AVGO", Corr: 0.5135700399870929},
			{Date: "2022-12-27", X0: "AAPL", X1: "GOOG", Corr: 0.19453030218238854},
			{Date: "2022-12-27", X0: "AAPL", X1: "INTC", Corr: 0.4403990244849937},
			{Date: "2022-12-27", X0: "AAPL", X1: "META", Corr: 0.39386893288119945},
			{Date: "2022-12-27", X0: "AAPL", X1: "MSFT", Corr: 0.4817176327224049},
			{Date: "2022-12-27", X0: "AAPL", X1: "NVDA", Corr: 0.5023715611927752},
			{Date: "2022-12-27", X0: "AAPL", X1: "QCOM", Corr: 0.5238656511138837},
			{Date: "2022-12-27", X0: "AAPL", X1: "TSLA", Corr: 0.5498852123024683},
			{Date: "2022-12-27", X0: "AMZN", X1: "AVGO", Corr: 0.7952426819328237},
			{Date: "2022-12-27", X0: "AMZN", X1: "GOOG", Corr: 0.35188586495461893},
			{Date: "2022-12-27", X0: "AMZN", X1: "INTC", Corr: 0.697882967539835},
			{Date: "2022-12-27", X0: "AMZN", X1: "META", Corr: 0.7055127904435734},
			{Date: "2022-12-27", X0: "AMZN", X1: "MSFT", Corr: 0.7019173408166158},
			{Date: "2022-12-27", X0: "AMZN", X1: "NVDA", Corr: 0.7260722394406184},
			{Date: "2022-12-27", X0: "AMZN", X1: "QCOM", Corr: 0.7272532605981774},
			{Date: "2022-12-27", X0: "AMZN", X1: "TSLA", Corr: 0.9104063921158937},
			{Date: "2022-12-27", X0: "AVGO", X1: "GOOG", Corr: 0.47587714174716406},
			{Date: "2022-12-27", X0: "AVGO", X1: "INTC", Corr: 0.8082912216745181},
			{Date: "2022-12-27", X0: "AVGO", X1: "META", Corr: 0.7746634797304436},
			{Date: "2022-12-27", X0: "AVGO", X1: "MSFT", Corr: 0.8602847216586795},
			{Date: "2022-12-27", X0: "AVGO", X1: "NVDA", Corr: 0.6377690437133635},
			{Date: "2022-12-27", X0: "AVGO", X1: "QCOM", Corr: 0.8515255277410169},
			{Date: "2022-12-27", X0: "AVGO", X1: "TSLA", Corr: 0.8289691320432666},
			{Date: "2022-12-27", X0: "GOOG", X1: "INTC", Corr: 0.44132420140211615},
			{Date: "2022-12-27", X0: "GOOG", X1: "META", Corr: 0.6184997480729949},
			{Date: "2022-12-27", X0: "GOOG", X1: "MSFT", Corr: 0.4135973438859086},
			{Date: "2022-12-27", X0: "GOOG", X1: "NVDA", Corr: 0.45615499528776976},
			{Date: "2022-12-27", X0: "GOOG", X1: "QCOM", Corr: 0.40408105956893564},
			{Date: "2022-12-27", X0: "GOOG", X1: "TSLA", Corr: 0.3646675474220342},
			{Date: "2022-12-27", X0: "INTC", X1: "META", Corr: 0.798152957264765},
			{Date: "2022-12-27", X0: "INTC", X1: "MSFT", Corr: 0.8165799823040059},
			{Date: "2022-12-27", X0: "INTC", X1: "NVDA", Corr: 0.6697022210513814},
			{Date: "2022-12-27", X0: "INTC", X1: "QCOM", Corr: 0.8765020479402509},
			{Date: "2022-12-27", X0: "INTC", X1: "TSLA", Corr: 0.7980828372890855},
			{Date: "2022-12-27", X0: "META", X1: "MSFT", Corr: 0.7756588319810561},
			{Date: "2022-12-27", X0: "META", X1: "NVDA", Corr: 0.713536049308839},
			{Date: "2022-12-27", X0: "META", X1: "QCOM", Corr: 0.7643043773538588},
			{Date: "2022-12-27", X0: "META", X1: "TSLA", Corr: 0.7926189782086763},
			{Date: "2022-12-27", X0: "MSFT", X1: "NVDA", Corr: 0.5045881485309515},
			{Date: "2022-12-27", X0: "MSFT", X1: "QCOM", Corr: 0.7834928000403768},
			{Date: "2022-12-27", X0: "MSFT", X1: "TSLA", Corr: 0.7688564829562652},
			{Date: "2022-12-27", X0: "NVDA", X1: "QCOM", Corr: 0.6256210476226092},
			{Date: "2022-12-27", X0: "NVDA", X1: "TSLA", Corr: 0.7394663446100521},
			{Date: "2022-12-27", X0: "QCOM", X1: "TSLA", Corr: 0.7849776694898971},
		},
		Date: []string{"2022-12-27", "2022-12-28"},
	}
	prefix := "dmag_d8K"
	value := db.User{
		EmailAddress: "test123@example.com",
		Prefix:       "dmag_d8K",
		Token:        "$2a$14$eIWUgPMqNQbpPveJdoQ8sOSw7DY5zBXUP3uUhm31LrfbArv6ZIhXe",
		GeneratedAt:  "2022-12-30 18:09:35",
		ExpiredAt:    "2023-06-30 18:09:35",
	}
	testCases := []struct {
		name          string
		token         string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, token string)
		checkResponse func(t *testing.T, recoder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			body: gin.H{
				"stocks":               []string{"AAPL", "AVGO", "TSLA"},
				"strike":               0.80,
				"autocall_coupon_rate": 0.50,
				"barrier_coupon_rate":  0.20,
				"fixed_coupon_rate":    0.20,
				"knock_out_barrier":    1.05,
				"knock_in_barrier":     0.70,
				"coupon_barrier":       0.80,
				"maturity":             12,
				"frequency":            3,
				"isEuro":               true,
			},
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
				store.EXPECT().GetBacktestValues(gomock.Any()).Times(1).Return(values, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "ERROR_BINDING",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			body: gin.H{
				"stocks":               []string{"AAPL", "AVGO", "TSLA"},
				"strike":               0,
				"autocall_coupon_rate": 0.50,
				"barrier_coupon_rate":  0.20,
				"fixed_coupon_rate":    0.20,
				"knock_out_barrier":    1.05,
				"knock_in_barrier":     0.70,
				"coupon_barrier":       0.80,
				"maturity":             12,
				"frequency":            3,
				"isEuro":               true,
			},
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
				store.EXPECT().GetBacktestValues(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "EMPTY_STOCK_LIST",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			body: gin.H{
				"stocks":               []string{},
				"strike":               0.80,
				"autocall_coupon_rate": 0.50,
				"barrier_coupon_rate":  0.20,
				"fixed_coupon_rate":    0.20,
				"knock_out_barrier":    1.05,
				"knock_in_barrier":     0.70,
				"coupon_barrier":       0.80,
				"maturity":             12,
				"frequency":            3,
				"isEuro":               true,
			},
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
				store.EXPECT().GetBacktestValues(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "MATURITY_LESS_THAN_FREQUENCY",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			body: gin.H{
				"stocks":               []string{"AAPL", "AVGO", "TSLA"},
				"strike":               0.80,
				"autocall_coupon_rate": 0.50,
				"barrier_coupon_rate":  0.20,
				"fixed_coupon_rate":    0.20,
				"knock_out_barrier":    1.05,
				"knock_in_barrier":     0.70,
				"coupon_barrier":       0.80,
				"maturity":             2,
				"frequency":            3,
				"isEuro":               true,
			},
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
				store.EXPECT().GetBacktestValues(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "FILTER_ERROR",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			body: gin.H{
				"stocks":               []string{"AA-PL", "A-VGO", "TSL-A"},
				"strike":               0.80,
				"autocall_coupon_rate": 0.50,
				"barrier_coupon_rate":  0.20,
				"fixed_coupon_rate":    0.20,
				"knock_out_barrier":    1.05,
				"knock_in_barrier":     0.70,
				"coupon_barrier":       0.80,
				"maturity":             12,
				"frequency":            3,
				"isEuro":               true,
			},
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
				store.EXPECT().GetBacktestValues(gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "EMPTY_RESULT",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			body: gin.H{
				"stocks":               []string{"AAPL", "AVGO", "TSLA"},
				"strike":               0.80,
				"autocall_coupon_rate": 0.50,
				"barrier_coupon_rate":  0.20,
				"fixed_coupon_rate":    0.20,
				"knock_out_barrier":    1.05,
				"knock_in_barrier":     0.70,
				"coupon_barrier":       0.80,
				"maturity":             12,
				"frequency":            3,
				"isEuro":               true,
			},
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
				store.EXPECT().GetBacktestValues(gomock.Any()).Times(1).Return(db.GetBacktestValuesResult{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:  "INTERNAL_SERVER_ERROR",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			body: gin.H{
				"stocks":               []string{"AAPL", "AVGO", "TSLA"},
				"strike":               0.80,
				"autocall_coupon_rate": 0.50,
				"barrier_coupon_rate":  0.20,
				"fixed_coupon_rate":    0.20,
				"knock_out_barrier":    1.05,
				"knock_in_barrier":     0.70,
				"coupon_barrier":       0.80,
				"maturity":             12,
				"frequency":            3,
				"isEuro":               true,
			},
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
				store.EXPECT().GetBacktestValues(gomock.Any()).Times(1).Return(db.GetBacktestValuesResult{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewServer(store)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/v1/backtest"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, tc.token)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestMinMax(t *testing.T) {
	type testCases struct {
		name  string
		array []float64
	}

	for _, scenario := range []testCases{
		{
			name:  "OK_1",
			array: []float64{0.2, 0.5, 0.6},
		},
		{
			name:  "OK_2",
			array: []float64{0.6, 0.5, 0.1},
		},
		{
			name:  "EQUAL",
			array: []float64{0.2, 0.2, 0.2},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			min, max := minmax(scenario.array)
			require.NotEmpty(t, min)
			require.NotEmpty(t, max)
			require.LessOrEqual(t, min, max)
		})
	}
}

func TestFCNPayout(t *testing.T) {
	stocks := []string{"AAPL", "AVGO", "TSLA"}
	arg1 := pricerRequest{
		Stocks:     []string{"AAPL", "AVGO", "TSLA"},
		Strike:     0.80,
		Cpn:        0.50,
		BarrierCpn: 0.50,
		FixCpn:     0.50,
		KO:         1.05,
		KI:         0.70,
		KC:         0.80,
		Maturity:   12,
		Freq:       3,
		IsEuro:     true,
	}
	arg2 := pricerRequest{
		Stocks:     []string{"AAPL", "AVGO", "TSLA"},
		Strike:     0.80,
		Cpn:        0.50,
		BarrierCpn: 0.50,
		FixCpn:     0.50,
		KO:         1.05,
		KI:         0.70,
		KC:         0.80,
		Maturity:   2,
		Freq:       3,
		IsEuro:     true,
	}
	fixing := map[string]float64{"AAPL": 130.03, "AVGO": 553.54, "TSLA": 109.1}
	mean := map[string]float64{"AAPL": -0.0024065238291240444, "AVGO": 0.0029074417269861117, "TSLA": -0.015126507431615293}
	px := map[string]float64{"AAPL": 130.03, "AVGO": 553.54, "TSLA": 109.1}
	models1 := map[string]mc.Model{
		"AAPL": mc.HypHyp{Sigma: 0.38956884573910466, Alpha: 0.31754204762725213, Beta: 0.09668058826922904, Kappa: 18.55196217354717, Rho: -0.08156231110497626},
		"AVGO": mc.HypHyp{Sigma: 0.33169818989315536, Alpha: 0.414590139046433, Beta: 0.4096664715295601, Kappa: 31.15386811469867, Rho: -0.2467638237838846},
		"TSLA": mc.HypHyp{Sigma: 0.926280232995074, Alpha: 0.09316279525141707, Beta: 0.11993430192118938, Kappa: 167.74229696983923, Rho: 0.9999999982622454},
	}
	corr1 := []float64{1.0, 0.5135700399870929, 0.5498852123024683, 0.5135700399870929, 1.0, 0.8289691320432666, 0.5498852123024683, 0.8289691320432666, 1.0}
	corr2 := []float64{-3.0, 2.0, 0.0, 2.0, -3.0, 0.0, 0.0, 0.0, -5.0}

	type testCases struct {
		name       string
		date       string
		stocks     []string
		arg        pricerRequest
		fixings    map[string]float64
		means      map[string]float64
		px         map[string]float64
		models     map[string]mc.Model
		corrMatrix *mat.SymDense
	}

	for _, test := range []testCases{
		{
			name:       "OK",
			date:       "2022-12-28",
			stocks:     stocks,
			arg:        arg1,
			fixings:    fixing,
			means:      mean,
			px:         px,
			models:     models1,
			corrMatrix: mat.NewSymDense(3, corr1),
		},
		{
			name:       "DISTRIBUTION_ERROR",
			date:       "2022-12-28",
			stocks:     stocks,
			arg:        arg1,
			fixings:    fixing,
			means:      mean,
			px:         px,
			models:     models1,
			corrMatrix: mat.NewSymDense(3, corr2),
		},
		{
			name:       "GENERATE_DATE_ERROR",
			date:       "2022-12-28",
			stocks:     stocks,
			arg:        arg2,
			fixings:    fixing,
			means:      mean,
			px:         px,
			models:     models1,
			corrMatrix: mat.NewSymDense(3, corr1),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			p, err := fcnPayout(test.date, test.stocks, test.arg, test.fixings, test.means, test.px, test.models, test.corrMatrix)
			t.Log(p)
			if test.name == "OK" {
				require.NoError(t, err)
				require.NotEmpty(t, p)
			} else {
				require.Error(t, err)
				require.Equal(t, true, math.IsNaN(p))
			}
		})
	}
}
