package handler

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBondList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		// rr := httptest.NewRecorder()
		// mockBondService := new(mocks.BondDelivery)
		// router := gin.Default()

		// NewHandler(&Config{
		// 	R:           router,
		// 	BondService: mockBondService,
		// })

		// reqBody, err := json.Marshal(gin.H{
		// 	"row":     10,
		// 	"page":    0,
		// 	"address": "12BnVhXxGBZXoq9QAkSv9UtVcdBs1k38yNx6sHUJWasTgYrm",
		// })

		// assert.NoError(t, err)
		// request, err := http.NewRequest(http.MethodPost, "/api/scan/bondlist", bytes.NewBuffer(reqBody))
		// request.Header.Set("Content-Type", "application/json")
		// assert.NoError(t, err)
		// router.ServeHTTP(rr, request)
		// assert.Equal(t, http.StatusOK, rr.Code)

		// below testing can't execute til plugin data model integrated into core subscan

		// mockUserResp := []b.Bond{{
		// 	ID:             1,
		// 	Account:        "12BnVhXxGBZXoq9QAkSv9UtVcdBs1k38yNx6sHUJWasTgYrm",
		// 	ExtrinsicIndex: "5095843-1",
		// }}

		// mockArgs := mock.Arguments{
		// 	10,
		// 	0,
		// 	"12BnVhXxGBZXoq9QAkSv9UtVcdBs1k38yNx6sHUJWasTgYrm",
		// }

		// mockBondService.On("BondList", mockArgs...).Return(mockUserResp, nil)
		// mockBondService.AssertCalled(t, "BondList", mockArgs...)
	})
}
