# ⚖️ Golang Order Matching Engine

A fully functional **order matching engine** built with **Go (Golang)**, supporting **limit** and **market** orders, symbol-based matching, and **MySQL** persistence. Designed to mimic core behavior of a trading system.

---

## 🔧 Features

- 📥 Place Limit and Market Orders
- 🔄 Automatic Matching Engine (Buy/Sell FIFO with price priority)
- 🪙 Symbol-based Order Book
- 💾 MySQL Persistence (Orders + Trades)
- 📊 Real-time Order Book & Trade Query APIs
- ❌ Cancel Orders
- ✅ Input Validation & Edge Case Handling

---

## 🛠️ Setup

### 1. Clone the Repo

```bash
git clone https://github.com/YOUR_USERNAME/order-matching-engine.git
cd order-matching-engine
```

### 2. Set Up MySQL

Run the following SQL to create the database and tables:

```sql
CREATE DATABASE IF NOT EXISTS order_matching;
USE order_matching;

DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS orders;

CREATE TABLE orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    type ENUM('limit', 'market') NOT NULL,
    side ENUM('buy', 'sell') NOT NULL,
    price DECIMAL(10,2),
    initial_quantity DECIMAL(10,2) NOT NULL,
    remaining_quantity DECIMAL(10,2) NOT NULL,
    status ENUM('open', 'partially_filled', 'filled', 'canceled') NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trades (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    buy_order_id BIGINT NOT NULL,
    sell_order_id BIGINT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    quantity DECIMAL(10,2) NOT NULL,
    traded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    symbol VARCHAR(10) NOT NULL DEFAULT 'XYZ',
    FOREIGN KEY (buy_order_id) REFERENCES orders(id),
    FOREIGN KEY (sell_order_id) REFERENCES orders(id)
);
```

### 3. Install Dependencies

```bash
go mod tidy
```

### 4. Run the App

```bash
go run cmd/main.go
```

Server will start on: [http://localhost:3000](http://localhost:3000)

---

## 📡 API Reference

### ✅ Health Check

```http
GET /health
```

---

### 🛒 Place Order

```http
POST /orders
```

#### JSON Body

```json
{
  "symbol": "XYZ",
  "type": "limit",
  "side": "buy",
  "price": 199,
  "quantity": 5
}
```

---

### 📘 Get Order Book

```http
GET /orderbook?symbol=XYZ
```

---

### 📄 Get All Orders

```http
GET /orders
```

---

### 🔍 Get Order Status

```http
GET /orders/{orderId}
```

---

### 🧾 Get Trades by Symbol

```http
GET /trades?symbol=XYZ
```

---

### ❌ Cancel Order

```http
DELETE /orders/{orderId}
```

---

## 🧪 Example curl

```bash
curl -X POST http://localhost:3000/orders -H "Content-Type: application/json" -d '{"symbol":"XYZ","type":"limit","side":"sell","price":199,"quantity":5}'
```

```bash
curl http://localhost:3000/orderbook?symbol=XYZ
```

---

## 📁 Project Structure

```
.
├── api/               # Handlers for HTTP endpoints
│   └── handlers.go
├── engine/            # Matching engine logic
│   └── matcher.go
├── models/            # Trade and Order structs
│   ├── orders.go
│   └── trades.go
├── cmd/
│   └── main.go        # App entrypoint
├── go.mod
├── go.sum
└── README.md
```

---

## 📄 License

```text
MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction...
```

---

## 🙌 Credits

Built for educational purposes. Extendable with WebSockets, Redis, metrics, and more.