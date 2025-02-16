package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/basedalex/merch-shop/internal/auth"
	"github.com/basedalex/merch-shop/internal/config"
	"github.com/basedalex/merch-shop/internal/db"
	"github.com/basedalex/merch-shop/internal/service"
	"github.com/go-playground/assert"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var testDB *pgxpool.Pool
var cfg *config.Config
var connect = "postgres://postgres:password@host.docker.internal:5433/merch-shop?sslmode=disable"

func TestMain(m *testing.M) {
	var err error

	cfg, _ = config.Init("../../config.dev.yaml")
	cfg.Database.DSN = connect
	cfg.Database.Migrations = "../migrations"

	fmt.Println(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo, _ := db.NewPostgres(ctx, cfg)
	service.NewService(repo)

	pool, err := pgxpool.New(ctx, connect)
	if err != nil {
		fmt.Println("Failed to connect to test database:", err)
	}
	testDB = pool

	resetTestDB(ctx)

	m.Run()

	testDB.Close()
}

func resetTestDB(ctx context.Context) {
	testDB.Exec(ctx, "DELETE FROM employee_purchases")
	testDB.Exec(ctx, "DELETE FROM employees")
	testDB.Exec(ctx, "DELETE FROM merch_shop")
}

func TestGetApiBuyItem(t *testing.T) {
	ctx := context.Background()

	repo, err := db.NewPostgres(ctx, cfg)
	require.NoError(t, err)
	s := service.NewService(repo)

	_, err = testDB.Exec(ctx, "INSERT INTO merch_shop (product_name, price) VALUES ('t-shirt', 100)")
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, "INSERT INTO employees (username, pass, balance) VALUES ('testuser', 'hashedpass', 200)")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/buy/t-shirt", nil)
	token, err := auth.CreateToken("testuser")
	require.NoError(t, err, "could not create token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.GetApiBuyItem(w, req, "t-shirt")

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updatedBalance int
	err = testDB.QueryRow(ctx, "SELECT balance FROM employees WHERE username = 'testuser'").Scan(&updatedBalance)
	require.NoError(t, err)
	assert.Equal(t, 100, updatedBalance)

	var purchaseCount int
	err = testDB.QueryRow(ctx, "SELECT COUNT(*) FROM employee_purchases WHERE employee_username = 'testuser' AND product_name = 't-shirt'").Scan(&purchaseCount)
	require.NoError(t, err)
	assert.Equal(t, 1, purchaseCount)
}

func TestPostApiSendCoin(t *testing.T) {
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `INSERT INTO employees (username, pass, balance) VALUES ('alice', 'hashedpass', 100), ('bob', 'hashedpass', 50)`)
	require.NoError(t, err, "error seeding users")

	repo, err := db.NewPostgres(ctx, cfg)
	require.NoError(t, err)
	s := service.NewService(repo)

	token, err := auth.CreateToken("alice")
	require.NoError(t, err, "could not create token")

	reqBody, err := json.Marshal(db.SendCoinRequest{
		ToUser: "bob",
		Amount: 30,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/sendCoin", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.PostApiSendCoin(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	var senderBalance, receiverBalance int
	err = testDB.QueryRow(ctx, "SELECT balance FROM employees WHERE username = 'alice'").Scan(&senderBalance)
	require.NoError(t, err)
	err = testDB.QueryRow(ctx, "SELECT balance FROM employees WHERE username = 'bob'").Scan(&receiverBalance)
	require.NoError(t, err)

	assert.Equal(t, 70, senderBalance)
	assert.Equal(t, 80, receiverBalance)

	var count int
	err = testDB.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE sender = 'alice' AND receiver = 'bob' AND amount = 30").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
