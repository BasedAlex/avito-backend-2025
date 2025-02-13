-- +goose Up
-- +goose StatementBegin
CREATE TABLE merch_shop (
    id BIGSERIAL PRIMARY KEY,
    product_name TEXT NOT NULL UNIQUE,
    price INT NOT NULL CHECK (price > 0),
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW()
);

CREATE TABLE employees (
    id BIGSERIAL PRIMARY KEY,
    balance INT NOT NULL DEFAULT 1000 CHECK (balance >= 0),
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW()
);

CREATE TABLE employee_merch (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    merch_id BIGINT NOT NULL REFERENCES merch_shop(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    purchased_at TIMESTAMP DEFAULT NOW()
)

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    receiver_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    amount INT NOT NULL CHECK (amount > 0),
    transaction_date TIMESTAMP DEFAULT NOW(),
    CHECK (sender_id <> receiver_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE merch_shop;
DROP TABLE employees;
DROP TABLE employee_merch;
DROP TABLE transactions;
-- +goose StatementEnd
