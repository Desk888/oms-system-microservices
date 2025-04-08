#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

API_URL="http://localhost:8080"

echo "Testing Order Management System API endpoints..."
echo "---------------------------------------------"

# Test User Service
echo -e "\n${GREEN}Testing User Service:${NC}"

# Create user
echo -e "\nCreating user..."
USER_RESPONSE=$(curl -s -X POST $API_URL/users -H "Content-Type: application/json" -d '{
  "email": "test@example.com",
  "password": "password123",
  "name": "Test User",
  "role": "user"
}')
USER_ID=$(echo $USER_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ ! -z "$USER_ID" ]; then
  echo -e "${GREEN}✓ Create user successful${NC}"
else
  echo -e "${RED}✗ Create user failed${NC}"
fi

# Authenticate user
echo -e "\nAuthenticating user..."
AUTH_RESPONSE=$(curl -s -X POST $API_URL/auth -H "Content-Type: application/json" -d '{
  "email": "test@example.com",
  "password": "password123"
}')
TOKEN=$(echo $AUTH_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ ! -z "$TOKEN" ]; then
  echo -e "${GREEN}✓ Authentication successful${NC}"
else
  echo -e "${RED}✗ Authentication failed${NC}"
fi

# Test Product Service
echo -e "\n${GREEN}Testing Product Service:${NC}"

# Create product
echo -e "\nCreating product..."
PRODUCT_RESPONSE=$(curl -s -X POST $API_URL/products -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{
  "name": "Test Product",
  "description": "A test product",
  "price": 99.99,
  "stock": 100
}')
PRODUCT_ID=$(echo $PRODUCT_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ ! -z "$PRODUCT_ID" ]; then
  echo -e "${GREEN}✓ Create product successful${NC}"
else
  echo -e "${RED}✗ Create product failed${NC}"
fi

# Update product stock
echo -e "\nUpdating product stock..."
STOCK_RESPONSE=$(curl -s -X PUT "$API_URL/products/$PRODUCT_ID/stock" -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{
  "quantity": 90
}')

if [[ $STOCK_RESPONSE == *"90"* ]]; then
  echo -e "${GREEN}✓ Update stock successful${NC}"
else
  echo -e "${RED}✗ Update stock failed${NC}"
fi

# Test Order Service
echo -e "\n${GREEN}Testing Order Service:${NC}"

# Create order
echo -e "\nCreating order..."
ORDER_RESPONSE=$(curl -s -X POST $API_URL/orders -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{
  "user_id": "'$USER_ID'",
  "items": [
    {
      "product_id": "'$PRODUCT_ID'",
      "quantity": 2,
      "price": 99.99
    }
  ]
}')
ORDER_ID=$(echo $ORDER_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ ! -z "$ORDER_ID" ]; then
  echo -e "${GREEN}✓ Create order successful${NC}"
else
  echo -e "${RED}✗ Create order failed${NC}"
fi

# Cleanup
echo -e "\n${GREEN}Cleaning up:${NC}"

# Delete order
echo -e "\nDeleting order..."
curl -s -X DELETE "$API_URL/orders/$ORDER_ID" -H "Authorization: Bearer $TOKEN" > /dev/null
echo -e "${GREEN}✓ Order deleted${NC}"

# Delete product
echo -e "\nDeleting product..."
curl -s -X DELETE "$API_URL/products/$PRODUCT_ID" -H "Authorization: Bearer $TOKEN" > /dev/null
echo -e "${GREEN}✓ Product deleted${NC}"

# Delete user
echo -e "\nDeleting user..."
curl -s -X DELETE "$API_URL/users/$USER_ID" -H "Authorization: Bearer $TOKEN" > /dev/null
echo -e "${GREEN}✓ User deleted${NC}"

echo -e "\n${GREEN}Tests completed!${NC}"
