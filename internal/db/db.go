package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/basedalex/merch-shop/internal/config"
	api "github.com/basedalex/merch-shop/internal/swagger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=db.go -destination=../mocks/mock_db.go -package=mocks
type Repository interface {
	GetEmployeeInfo(ctx context.Context, employeeName string) (*InfoResponse, error)
	TransferCoins(ctx context.Context, senderName, receiverName string, amount int) error
	BuyItem(ctx context.Context, employeeName, item string) error
	Authenticate(ctx context.Context, authRequest api.AuthRequest) (bool, error)
	CreateEmployee(ctx context.Context, authRequest api.AuthRequest) error
}

type Postgres struct {
	db *pgxpool.Pool
}

func NewPostgres(ctx context.Context, cfg *config.Config) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("error parsing connection string: %w", err)
	}

	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("error pinging the database: %w", err)
	}

	if err := runMigrations(db, cfg.Database.Migrations); err != nil {
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	return &Postgres{db: db}, nil
}

func runMigrations(db *pgxpool.Pool, path string) error {
	sqlDB := stdlib.OpenDBFromPool(db)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(sqlDB, path); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close SQL DB: %w", err)
	}

	return nil
}

func (p *Postgres) GetEmployeeInfo(ctx context.Context, employeeName string) (*InfoResponse, error) {
	var info InfoResponse

	query := `SELECT product_name, SUM(1) AS quantity FROM employee_purchases WHERE employee_username = $1 GROUP BY product_name`

	rows, err := p.db.Query(ctx, query, employeeName)
	if err != nil {
		return nil, fmt.Errorf("error fetching employee info: %w", err)
	}
	for rows.Next() {
		var productName string
		var quantity int

		err := rows.Scan(&productName, &quantity)
		if err != nil {
			return nil, fmt.Errorf("error fetching employee info: %w", err)
		}

		info.Inventory = append(info.Inventory, Item{Type: productName, Quantity: quantity})
	}
	query = `SELECT balance FROM employees WHERE username = $1`

	if err := p.db.QueryRow(ctx, query, employeeName).Scan(&info.Coins); err != nil {
		return nil, fmt.Errorf("error fetching employee info: %w", err)
	}

	query = `SELECT sender, receiver, amount, transaction_date FROM transactions WHERE receiver = $1;`

	rows, err = p.db.Query(ctx, query, employeeName)
	if err != nil {
		return nil, fmt.Errorf("error fetching employee transactions: %w", err)
	}
	for rows.Next() {
		var sender, receiver string
		var amount int
		var transactionDate time.Time

		err := rows.Scan(&sender, &receiver, &amount, &transactionDate)
		if err != nil {
			return nil, fmt.Errorf("error fetching employee info: %w", err)
		}

		info.CoinHistory.Received = append(info.CoinHistory.Received, Transaction{
			FromUser:        sender,
			ToUser:          receiver,
			Amount:          amount,
			TransactionDate: transactionDate,
		})
	}


	query = `SELECT sender, receiver, amount, transaction_date FROM transactions WHERE sender = $1;`

	rows, err = p.db.Query(ctx, query, employeeName)
	if err != nil {
		return nil, fmt.Errorf("error fetching employee transactions: %w", err)
	}
	for rows.Next() {
		var sender, receiver string
		var amount int
		var transactionDate time.Time

		err := rows.Scan(&sender, &receiver, &amount, &transactionDate)
		if err != nil {
			return nil, fmt.Errorf("error fetching employee info: %w", err)
		}

		info.CoinHistory.Sent = append(info.CoinHistory.Sent, Transaction{
			FromUser:        sender,
			ToUser:          receiver,
			Amount:          amount,
			TransactionDate: transactionDate,
		})
	}

	return &info, nil
}

func (p *Postgres) TransferCoins(ctx context.Context, sender, receiver string, amount int) (err error) {
	if amount <= 0 {
		return fmt.Errorf("invalid transfer amount: %d", amount)
	}
	if sender == receiver {
		return fmt.Errorf("sender and receiver cannot be the same")
	}

	if sender > receiver {
		sender, receiver = receiver, sender
		amount = -amount
	}

	tx, err := p.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var fromBalance int
	err = tx.QueryRow(ctx, `SELECT balance FROM employees WHERE username=$1 FOR UPDATE;`, sender).Scan(&fromBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("sender not found")
		}
		return err
	}

	var toBalance int
	err = tx.QueryRow(ctx, `SELECT balance FROM employees WHERE username=$1 FOR UPDATE`, receiver).Scan(&toBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("sender not found")
		}
		return err
	}

	fromBalance -= amount
	toBalance += amount
	if fromBalance < 0 || toBalance < 0 {
		return fmt.Errorf("not enough money on balance")
	}

	query := "UPDATE employees SET balance=$1 WHERE username=$2"

	_, err = tx.Exec(ctx, query, fromBalance, sender)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, query, toBalance, receiver)
	if err != nil {
		return err
	}

	if amount < 0 {
		sender, receiver = receiver, sender
		amount = -amount
	}

	query = `INSERT INTO transactions (sender, receiver, amount) VALUES ($1, $2, $3)`
	_, err = tx.Exec(ctx, query, sender, receiver, amount)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	return err
}

func (p *Postgres) BuyItem(ctx context.Context, employeeName, item string) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var price int
	if err := tx.QueryRow(ctx, `SELECT price FROM merch_shop WHERE product_name = $1`, item).Scan(&price); err != nil {
		return fmt.Errorf("error getting item price: %w", err)
	}

	var balance int
	if err := tx.QueryRow(ctx, `SELECT balance FROM employees WHERE username = $1 FOR UPDATE`, employeeName).Scan(&balance); err != nil {
		return fmt.Errorf("error fetching balance: %w", err)
	}
	if balance < price {
		return fmt.Errorf("not enough balance")
	}

	_, err = tx.Exec(ctx, `UPDATE employees SET balance = balance - $1 WHERE username = $2`, price, employeeName)
	if err != nil {
		return fmt.Errorf("error deducting coins for purchase: %w", err)
	}

	query := `INSERT INTO employee_purchases (employee_username, product_name) VALUES ($1, $2)`
	_, err = tx.Exec(ctx, query, employeeName, item)
	if err != nil {
		return fmt.Errorf("error inserting purchase record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

func (p *Postgres) Authenticate(ctx context.Context, authRequest api.AuthRequest) (bool, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var pass string
	err = tx.QueryRow(ctx, `SELECT pass FROM employees WHERE username = $1;`, authRequest.Username).Scan(&pass)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("error getting user password: %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(pass), []byte(authRequest.Password)); err != nil {
		return true, err
	}

	return true, nil
}

func (p *Postgres) CreateEmployee(ctx context.Context, authRequest api.AuthRequest) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var username string
	err = tx.QueryRow(ctx, `INSERT INTO employees (username, pass) VALUES ($1, $2) RETURNING username;`,
		authRequest.Username, authRequest.Password).Scan(&username)
	if err != nil {
		return fmt.Errorf("could not create new user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}
