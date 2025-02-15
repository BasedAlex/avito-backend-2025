
CREATE TABLE IF NOT EXISTS merch_shop (
    id BIGSERIAL PRIMARY KEY,
    product_name TEXT NOT NULL,
    price INT NOT NULL,
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW(),
);


INSERT INTO merch_shop (product_name, price) VALUES ('t-shirt', 80);

INSERT INTO merch_shop (product_name, price) VALUES ('cup', 20);

INSERT INTO merch_shop (product_name, price) 
VALUES ('book', 50);

INSERT INTO merch_shop (product_name, price) 
VALUES ('pen', 10);

INSERT INTO merch_shop (product_name, price) 
VALUES ('powerbank', 200);

INSERT INTO merch_shop (product_name, price) 
VALUES ('hoody', 300);

INSERT INTO merch_shop (product_name, price) 
VALUES ('umbrella', 200);

INSERT INTO merch_shop (product_name, price) 
VALUES ('socks', 10);

INSERT INTO merch_shop (product_name, price) 
VALUES ('wallet', 50);

INSERT INTO merch_shop (product_name, price) 
VALUES ('pink-hoody', 500);