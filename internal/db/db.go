package db

import (
	"context"
	"fmt"

	api "github.com/basedalex/merch-shop/internal/swagger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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


func (p *Postgres) TransferCoins(ctx context.Context, senderID, receiverID, amount int) (err error) {
	if amount <= 0 {
		return fmt.Errorf("invalid transfer amount: %d", amount)
	}
	if senderID == receiverID {
		return fmt.Errorf("sender and receiver cannot be the same")
	}

	fmt.Println("transfer coins started!")

	tx, err := p.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var fromBalance int
	err = tx.QueryRow(ctx, `SELECT balance FROM employees WHERE id=$1 FOR UPDATE;`, senderID).Scan(&fromBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("sender not found")
		}
		return err
	}
	if fromBalance < amount {
		return fmt.Errorf("not enough money on balance")
	}

	var toBalance int
	err = tx.QueryRow(ctx, `SELECT balance FROM employees WHERE id=$1 FOR UPDATE;`, receiverID).Scan(&toBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("receiver not found")
		}
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE employees 
        SET balance = CASE 
            WHEN id = $1 THEN balance - $3 
            WHEN id = $2 THEN balance + $3 
        END 
        WHERE id IN ($1, $2);`, senderID, receiverID, amount)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	return err
}


func (p *Postgres) BuyItem(ctx context.Context, employeeID int, item string) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var merchID, price int
	if err := tx.QueryRow(ctx, `SELECT id, price FROM merch_shop WHERE product_name = $1 FOR UPDATE;`, item).Scan(&merchID, &price); err != nil {
		return fmt.Errorf("error getting item price: %w", err)
	}

	var balance int
	if err := tx.QueryRow(ctx, `SELECT balance FROM employees WHERE id = $1 FOR UPDATE;`, employeeID).Scan(&balance); err != nil {
		return fmt.Errorf("error fetching balance: %w", err)
	}
	if balance < price {
		return fmt.Errorf("not enough balance")
	}

	_, err = tx.Exec(ctx, `UPDATE employees SET balance = balance - $1 WHERE id = $2;`, price, employeeID)
	if err != nil {
		return fmt.Errorf("error deducting coins for purchase: %w", err)
	}

	var quantity int
	err = tx.QueryRow(ctx, `SELECT quantity FROM employee_merch WHERE employee_id = $1 AND merch_id = $2 FOR UPDATE;`, employeeID, merchID).Scan(&quantity)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("error locking merch record: %w", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO employee_merch (employee_id, merch_id, quantity) 
		VALUES ($1, $2, 1) 
		ON CONFLICT (employee_id, merch_id) 
		DO UPDATE SET quantity = employee_merch.quantity + EXCLUDED.quantity;`, employeeID, merchID)
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
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	
	var pass string
	err = tx.QueryRow(ctx, `SELECT pass FROM employees WHERE username = $1;`, authRequest.Username).Scan(&pass)
	if err != nil {
		if err == pgx.ErrNoRows {
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
		if pErr := recover(); pErr != nil {
			_ = tx.Rollback(ctx)
			panic(pErr)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var employeeID int
	err = tx.QueryRow(ctx, `INSERT INTO employees (username, pass) VALUES ($1, $2) RETURNING id;`, 
		authRequest.Username, authRequest.Password).Scan(&employeeID)
	if err != nil {
		return fmt.Errorf("could not create new user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

func (p *Postgres) GetEmployeeID(ctx context.Context, username string) (int, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var ID int
	err = tx.QueryRow(ctx, `SELECT id FROM employees WHERE username = $1;`, username).Scan(&ID);
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("user with username %s not found", username)
		}
		return 0, fmt.Errorf("could not get user_id: %w", err)
	}


	return ID, nil
}