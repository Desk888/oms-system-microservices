package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pb "github.com/order-management/proto"
)

// Product handlers
func (g *APIGateway) createProduct(c *gin.Context) {
	var req pb.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := g.productClient.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func (g *APIGateway) getProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id is required"})
		return
	}

	product, err := g.productClient.GetProduct(c.Request.Context(), &pb.GetProductRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (g *APIGateway) updateProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id is required"})
		return
	}

	var req pb.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Id = id

	product, err := g.productClient.UpdateProduct(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (g *APIGateway) updateStock(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id is required"})
		return
	}

	var req struct {
		Quantity int32 `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call the product service to update stock
	product, err := g.productClient.UpdateStock(c.Request.Context(), &pb.UpdateStockRequest{
		Id:             id,
		QuantityChange: req.Quantity,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return a simplified response that includes the stock quantity
	c.JSON(http.StatusOK, gin.H{
		"id":             product.Id,
		"name":           product.Name,
		"stock_quantity": product.StockQuantity,
	})
}

func (g *APIGateway) listProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	category := c.Query("category")

	resp, err := g.productClient.ListProducts(c.Request.Context(), &pb.ListProductsRequest{
		Page:     int32(page),
		Limit:    int32(limit),
		Category: category,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
