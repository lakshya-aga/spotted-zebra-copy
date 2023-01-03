package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/banachtech/spotted-zebra/db/mock"
	db "github.com/banachtech/spotted-zebra/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	prefix := "dmag_d8K"
	value := db.User{
		EmailAddress: "test123@example.com",
		Prefix:       "dmag_d8K",
		Token:        "$2a$14$eIWUgPMqNQbpPveJdoQ8sOSw7DY5zBXUP3uUhm31LrfbArv6ZIhXe",
		GeneratedAt:  "2022-12-30 18:09:35",
		ExpiredAt:    "2023-06-30 18:09:35",
	}
	value2 := db.User{
		EmailAddress: "test123@example.com",
		Prefix:       "dmag_d8K",
		Token:        "$2a$14$eIWUgPMqNQbpPveJdoQ8sOSw7DY5zBXUP3uUhm31LrfbArv6ZIhXe",
		GeneratedAt:  "2022-11-30 18:09:35",
		ExpiredAt:    "2022-12-05 18:09:35",
	}
	testCases := []struct {
		name          string
		token         string
		date          string
		buildStubs    func(store *mockdb.MockStore)
		setupAuth     func(t *testing.T, request *http.Request, token string)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "NO_AUTHORIZATION",
			token: "",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:  "UNSUPPORTED_AUTHORIZATION",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", "unsupported", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:  "INVALID_AUTHORIZATION_FORMAT",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", "", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:  "EXPIRED_TOKEN",
			token: "dmag_d8K.RGbV3hb3LEwYohYW",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value2, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:  "WRONG_PREFIX_LENGTH",
			token: "dmag_d8.RGbV3hb3LEwYohYW",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:  "WRONG_API_KEY",
			token: "dmag_d8K.RGbV3hb3LEwYohYX",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(value, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:  "USER_NOT_EXISTS",
			token: "dmag_d8K.RGbV3hb3LEwYohYX",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(db.User{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:  "INTERNAL_SERVER_ERROR",
			token: "dmag_d8K.RGbV3hb3LEwYohYXX",
			setupAuth: func(t *testing.T, request *http.Request, token string) {
				authorizationHeader := fmt.Sprintf("%s %s", authorizationTypeBearer, token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(prefix)).Times(1).Return(db.User{}, sql.ErrConnDone)
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

			authPath := "/auth"
			server.router.GET(
				authPath,
				server.Authentication,
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, tc.token)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
