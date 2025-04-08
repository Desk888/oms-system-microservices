# Order Management System

A microservices-based order management system built with Go, MongoDB, gRPC, and REST APIs.

## Architecture

The system consists of the following microservices:
- **API Gateway**: Handles external REST API requests and routes them to appropriate services
- **Order Service**: Manages order creation, updates, and lifecycle
- **Product Service**: Handles product inventory and details
- **User Service**: Manages user authentication and profiles

## Tech Stack
- Go 1.21+
- MongoDB
- gRPC
- REST API
- Docker

## Project Structure
```
.
├── api-gateway/       # API Gateway service
├── order-service/     # Order management service
├── product-service/   # Product management service
├── user-service/      # User management service
├── proto/            # Protocol buffer definitions
└── pkg/              # Shared packages
```

## Getting Started

### Prerequisites
- Docker
- Docker Compose

### Running with Docker
1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd order-management-system
   ```

2. Start all services:
   ```bash
   docker-compose up --build
   ```

   This will start:
   - MongoDB on port 27017
   - Order Service on port 50051
   - Product Service on port 50052
   - User Service on port 50053
   - API Gateway on port 8080

3. Access the API Gateway at `http://localhost:8080`

## API Endpoints

### Orders
- POST `/orders` - Create an order
- GET `/orders/:id` - Get an order
- PUT `/orders/:id` - Update an order
- GET `/orders` - List orders

### Products
- POST `/products` - Create a product
- GET `/products/:id` - Get a product
- PUT `/products/:id` - Update a product
- GET `/products` - List products
- PUT `/products/:id/stock` - Update product stock

### Users
- POST `/users` - Create a user
- GET `/users/:id` - Get a user
- PUT `/users/:id` - Update a user
- DELETE `/users/:id` - Delete a user
- GET `/users` - List users
- POST `/auth` - Authenticate user

## Environment Variables

### API Gateway
- `ORDER_SERVICE_URL` - Order service URL (default: localhost:50051)
- `PRODUCT_SERVICE_URL` - Product service URL (default: localhost:50052)
- `USER_SERVICE_URL` - User service URL (default: localhost:50053)

### Services
- `MONGO_URI` - MongoDB connection URI (default: mongodb://localhost:27017)
- `JWT_SECRET` - Secret key for JWT tokens (User Service only)

## Development

### Local Development
Each service can be developed and deployed independently. Inter-service communication is handled via gRPC.

To run services locally:

1. Start MongoDB:
   ```bash
   docker-compose up mongodb
   ```

2. Run each service:
   ```bash
   cd order-service && go run main.go
   cd product-service && go run main.go
   cd user-service && go run main.go
   cd api-gateway && go run main.go
   ```

### Building Docker Images

To build all services:
```bash
docker-compose build
```

Or build individual services:
```bash
docker-compose build order-service
docker-compose build product-service
docker-compose build user-service
docker-compose build api-gateway
```
