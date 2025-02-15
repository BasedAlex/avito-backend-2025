-- +goose Up
-- +goose StatementBegin
CREATE TABLE merch_shop (
    product_name TEXT PRIMARY KEY,
    price INT NOT NULL CHECK (price > 0),
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW()
);

CREATE TABLE employees (
    username TEXT PRIMARY KEY,
    pass TEXT NOT NULL,
    balance INT NOT NULL DEFAULT 1000 CHECK (balance >= 0),
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW()
);

CREATE TABLE employee_purchases (
    id BIGSERIAL PRIMARY KEY,
    employee_username TEXT NOT NULL REFERENCES employees(username) ON DELETE CASCADE,
    product_name TEXT NOT NULL REFERENCES merch_shop(product_name) ON DELETE CASCADE,
    purchased_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    sender TEXT NOT NULL REFERENCES employees(username) ON DELETE CASCADE,
    receiver TEXT NOT NULL REFERENCES employees(username) ON DELETE CASCADE,
    amount INT NOT NULL CHECK (amount > 0),
    transaction_date TIMESTAMP DEFAULT NOW(),
    CHECK (sender <> receiver)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE merch_shop;
DROP TABLE employees;
DROP TABLE employee_purchases;
DROP TABLE transactions;
-- +goose StatementEnd
