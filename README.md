# Golang Order Matching System

## Setup

1. Install Go (>=1.20) and MySQL
2. Clone the repo: `git clone <repo_url> && cd golang-order-matching`
3. Run `go mod tidy`
4. Execute `db/sql_scripts.sql` in your MySQL instance
5. Update DB connection settings in your code/config
6. Start the server: `go run cmd/main.go`

## API Endpoints

- `GET /health`
- `POST /orders`
- `DELETE /orders/{id}`
- `GET /orderbook`

## Demo

Run the demo:
```bash
bash scripts/demo.sh
```
"# GOLANG-ORDER-MATCHING-SYSTEM" 
