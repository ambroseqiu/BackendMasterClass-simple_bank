package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/backendmaster/simple_bank/db/mock"
	db "github.com/backendmaster/simple_bank/db/sqlc"
	"github.com/backendmaster/simple_bank/token"
	"github.com/backendmaster/simple_bank/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestTransferAPI(t *testing.T) {
	amount := int64(10)

	user1, _ := randomUser(t)
	user2, _ := randomUser(t)
	user3, _ := randomUser(t)

	sameCurrencyAccount1 := randomAccount(user1.Username)
	sameCurrencyAccount2 := randomAccount(user2.Username)
	differentCurrencyAccount := randomAccount(user3.Username)
	sameCurrencyAccount1.Currency = util.USD
	sameCurrencyAccount2.Currency = util.USD
	differentCurrencyAccount.Currency = util.EUR

	testCase := []struct {
		name          string
		body          gin.H
		addAuth       func(request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "ok",
			body: gin.H{
				"from_account_id": sameCurrencyAccount1.ID,
				"to_account_id":   sameCurrencyAccount2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			addAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, sameCurrencyAccount1.Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(sameCurrencyAccount1.ID)).Times(1).Return(sameCurrencyAccount1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(sameCurrencyAccount2.ID)).Times(1).Return(sameCurrencyAccount2, nil)
				arg := db.TransferTxParams{
					FromAccountID: sameCurrencyAccount1.ID,
					ToAccountID:   sameCurrencyAccount2.ID,
					Amount:        amount,
				}
				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(arg)).Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "Invalid json body",
			body: gin.H{
				"from_account_id": -1,
			},
			addAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, sameCurrencyAccount1.Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Currency Mismatched of Account1",
			body: gin.H{
				"from_account_id": differentCurrencyAccount.ID,
				"to_account_id":   sameCurrencyAccount2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			addAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, sameCurrencyAccount1.Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(differentCurrencyAccount.ID)).Times(1).
					Return(differentCurrencyAccount, nil)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Currency Mismatched of Account2",
			body: gin.H{
				"from_account_id": sameCurrencyAccount1.ID,
				"to_account_id":   differentCurrencyAccount.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			addAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, sameCurrencyAccount1.Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(sameCurrencyAccount1.ID)).Times(1).
					Return(sameCurrencyAccount1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(differentCurrencyAccount.ID)).Times(1).
					Return(differentCurrencyAccount, nil)
				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Account1 Not Found",
			body: gin.H{
				"from_account_id": sameCurrencyAccount1.ID,
				"to_account_id":   sameCurrencyAccount2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			addAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, sameCurrencyAccount1.Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(sameCurrencyAccount1.ID)).Times(1).
					Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "Transcation Failed",
			body: gin.H{
				"from_account_id": sameCurrencyAccount1.ID,
				"to_account_id":   sameCurrencyAccount2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			addAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, sameCurrencyAccount1.Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(sameCurrencyAccount1.ID)).Times(1).Return(sameCurrencyAccount1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(sameCurrencyAccount2.ID)).Times(1).Return(sameCurrencyAccount2, nil)
				arg := db.TransferTxParams{
					FromAccountID: sameCurrencyAccount1.ID,
					ToAccountID:   sameCurrencyAccount2.ID,
					Amount:        amount,
				}
				store.EXPECT().TransferTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(db.TransferTxResult{}, sql.ErrTxDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Get Account failed",
			body: gin.H{
				"from_account_id": sameCurrencyAccount1.ID,
				"to_account_id":   sameCurrencyAccount2.ID,
				"amount":          amount,
				"currency":        util.USD,
			},
			addAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, sameCurrencyAccount1.Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(sameCurrencyAccount1.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCase {
		tc := testCase[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)

			//build stub
			tc.buildStubs(store)
			//start new server and send request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			//marshal body to json
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)
			url := "/transfers"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)
			tc.addAuth(request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			//check response
			tc.checkResponse(t, recorder)
		})
	}

}
