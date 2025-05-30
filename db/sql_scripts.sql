-- SQL migration scripts
CREATE DATABASE IF NOT EXISTS order_matching;
USE order_matching;

CREATE TABLE orders (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    symbol VARCHAR(10) NOT NULL,
    type VARCHAR(10) NOT NULL,
    side VARCHAR(4) NOT NULL,
    price DECIMAL(18,8),
    initial_quantity DECIMAL(18,8) NOT NULL,
    remaining_quantity DECIMAL(18,8) NOT NULL,
    status VARCHAR(10) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trades (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    buy_order_id BIGINT NOT NULL,
    sell_order_id BIGINT NOT NULL,
    price DECIMAL(18,8) NOT NULL,
    quantity DECIMAL(18,8) NOT NULL,
    traded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

