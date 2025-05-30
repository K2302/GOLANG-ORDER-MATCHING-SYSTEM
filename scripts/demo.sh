#!/bin/bash
# Demo script for testing the API

curl -X GET http://localhost:3000/health
curl -X POST http://localhost:3000/orders -d '{}'
