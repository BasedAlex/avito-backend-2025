package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/basedalex/merch-shop/internal/auth"
	"github.com/basedalex/merch-shop/internal/mocks"
	api "github.com/basedalex/merch-shop/internal/swagger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPostApiAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockRepository(ctrl)
	s := &MyService{db: mockDB}

	t.Run("Authenticate success, return token", func(t *testing.T) {
		authReq := api.AuthRequest{Username: "testuser", Password: "password"}
		requestBody, _ := json.Marshal(authReq)
		mockDB.EXPECT().Authenticate(gomock.Any(), authReq).Return(true, nil)
		token, _ := auth.CreateToken(authReq.Username)

		req := httptest.NewRequest(http.MethodPost, "/api/auth", bytes.NewBuffer(requestBody))
		w := httptest.NewRecorder()

		s.PostApiAuth(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, token, w.Body.String())
	})

	t.Run("Authenticate fail, create user and return token", func(t *testing.T) {
		authReq := api.AuthRequest{Username: "newuser", Password: "password"}
		requestBody, _ := json.Marshal(authReq)
		mockDB.EXPECT().Authenticate(gomock.Any(), authReq).Return(false, nil)
		mockDB.EXPECT().CreateEmployee(gomock.Any(), gomock.Any()).Return(nil)
		token, _ := auth.CreateToken(authReq.Username)

		req := httptest.NewRequest(http.MethodPost, "/api/auth", bytes.NewBuffer(requestBody))
		w := httptest.NewRecorder()

		s.PostApiAuth(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, token, w.Body.String())
	})

	t.Run("Authentication error", func(t *testing.T) {
		authReq := api.AuthRequest{Username: "user", Password: "wrongpass"}
		requestBody, _ := json.Marshal(authReq)

		mockDB.EXPECT().Authenticate(gomock.Any(), authReq).Return(true, errors.New("error: credentials are incorrect"))
		mockDB.EXPECT().CreateEmployee(gomock.Any(), gomock.Any()).Times(0)

		req := httptest.NewRequest(http.MethodPost, "/api/auth", bytes.NewBuffer(requestBody))
		w := httptest.NewRecorder()

		s.PostApiAuth(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "error: credentials are incorrect")
	})
}

func TestGetApiBuyItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockRepository(ctrl)
	s := NewService(mockDB)

	t.Run("Buy success", func(t *testing.T) {
		username := "test"
		item := "book"

		token, err := auth.CreateToken(username)
		assert.NoError(t, err)

		mockDB.EXPECT().BuyItem(gomock.Any(), username, item).Return(nil)

		req := httptest.NewRequest(http.MethodGet, "/api/buy/"+item, nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		w := httptest.NewRecorder()

		s.GetApiBuyItem(w, req, item)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "")
	})

}
