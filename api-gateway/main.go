package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "github.com/order-management/proto"
)

type APIGateway struct {
	orderClient   pb.OrderServiceClient
	productClient pb.ProductServiceClient
	userClient    pb.UserServiceClient
}

func main() {
	// Get service URLs from environment variables
	orderServiceURL := os.Getenv("ORDER_SERVICE_URL")
	if orderServiceURL == "" {
		orderServiceURL = "localhost:50051"
	}

	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		productServiceURL = "localhost:50052"
	}

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "localhost:50053"
	}

	// Connect to Order Service
	orderConn, err := grpc.Dial(orderServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Order service: %v", err)
	}
	defer orderConn.Close()

	// Connect to Product Service
	productConn, err := grpc.Dial(productServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Product service: %v", err)
	}
	defer productConn.Close()

	// Connect to User Service
	userConn, err := grpc.Dial(userServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to User service: %v", err)
	}
	defer userConn.Close()

	gateway := &APIGateway{
		orderClient:   pb.NewOrderServiceClient(orderConn),
		productClient: pb.NewProductServiceClient(productConn),
		userClient:    pb.NewUserServiceClient(userConn),
	}

	// Initialize Gin router
	r := gin.Default()

	// Order endpoints
	r.POST("/orders", gateway.createOrder)
	r.GET("/orders/:id", gateway.getOrder)
	r.PUT("/orders/:id", gateway.updateOrder)
	r.GET("/orders", gateway.listOrders)

	// Product endpoints
	r.POST("/products", gateway.createProduct)
	r.GET("/products/:id", gateway.getProduct)
	r.PUT("/products/:id", gateway.updateProduct)
	r.GET("/products", gateway.listProducts)
	r.PUT("/products/:id/stock", gateway.updateStock)

	// User endpoints
	r.POST("/users", gateway.createUser)
	r.GET("/users/:id", gateway.getUser)
	r.PUT("/users/:id", gateway.updateUser)
	r.DELETE("/users/:id", gateway.deleteUser)
	r.GET("/users", gateway.listUsers)
	r.POST("/auth", gateway.authenticateUser)

	// Start server
	log.Printf("API Gateway listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func (g *APIGateway) createOrder(c *gin.Context) {
	var req pb.CreateOrderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := g.orderClient.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (g *APIGateway) getOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := g.orderClient.GetOrder(c.Request.Context(), &pb.GetOrderRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (g *APIGateway) updateOrder(c *gin.Context) {
	id := c.Param("id")
	var req pb.UpdateOrderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Id = id

	order, err := g.orderClient.UpdateOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (g *APIGateway) listOrders(c *gin.Context) {
	var req pb.ListOrdersRequest
	if err := c.BindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := g.orderClient.ListOrders(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}


