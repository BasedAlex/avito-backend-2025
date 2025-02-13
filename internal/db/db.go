package db

import (
	"context"
	"fmt"

	"github.com/basedalex/merch-shop/internal/api"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetEmployeeInfo(ctx context.Context, employeeID int) (*InfoResponse, error)
	TransferCoins(ctx context.Context, senderID, receiverID, amount int) error
	BuyItem(ctx context.Context, employeeID int, item string) error
	Authenticate(ctx context.Context, authRequest api.AuthRequest) (bool, error)
	CreateEmployee(ctx context.Context, authRequest api.AuthRequest) error
	GetEmployeeID(ctx context.Context, username string) (int, error) 
}

type Postgres struct {
	db *pgxpool.Pool
}

func NewPostgres(ctx context.Context, conn string) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(conn)
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

	return &Postgres{db: db}, nil
}

func (p *Postgres) GetEmployeeInfo(ctx context.Context, employeeID int) (*InfoResponse, error) {
	var info InfoResponse
	query := `SELECT balance FROM employees WHERE id = $1;`
	
	if err := p.db.QueryRow(ctx, query, employeeID).Scan(&info.Coins); err != nil {
		return nil, fmt.Errorf("error fetching employee info: %w", err)
	}
	return &info, nil
}

func (p *Postgres) TransferCoins(ctx context.Context, senderID, receiverID, amount int) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE employees SET balance = balance - $1 WHERE id = $2 AND balance >= $1;`, amount, senderID)
	if err != nil {
		return fmt.Errorf("error deducting coins from sender: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE employees SET balance = balance + $1 WHERE id = $2;`, amount, receiverID)
	if err != nil {
		return fmt.Errorf("error adding coins to receiver: %w", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO transactions (sender_id, receiver_id, amount) VALUES ($1, $2, $3);`, senderID, receiverID, amount)
	if err != nil {
		return fmt.Errorf("error inserting transaction record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}


func (p *Postgres) BuyItem(ctx context.Context, employeeID int, item string) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	merchID, price := 0, 0
	if err := tx.QueryRow(ctx, `SELECT id, price FROM merch_shop WHERE product_name = $1;`, item).Scan(&merchID, price); err != nil {
		return fmt.Errorf("error getting item price: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE employees SET balance = balance - $1 WHERE id = $2 AND balance >= $1;`, price, employeeID)
	if err != nil {
		return fmt.Errorf("error deducting coins for purchase: %w", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO employee_merch (employee_id, merch_id) VALUES ($1, $2) ON CONFLICT (employee_id, merch_id) DO UPDATE SET quantity = employee_merch.quantity + 1;`, employeeID, merchID)
	if err != nil {
		return fmt.Errorf("error inserting purchase record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

func (p *Postgres) Authenticate(ctx context.Context, authRequest api.AuthRequest) (bool, error) {
	return true, nil
}

func (p *Postgres) CreateEmployee(ctx context.Context, authRequest api.AuthRequest) error {
	return nil
}

func (p *Postgres) GetEmployeeID(ctx context.Context, username string) (int, error) {
	return 1, nil
}