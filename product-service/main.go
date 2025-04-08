package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "github.com/order-management/proto"
)

type server struct {
	pb.UnimplementedProductServiceServer
	db *mongo.Database
}

func main() {
	// MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":50052") // Different port from order service
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterProductServiceServer(s, &server{
		db: client.Database("order_management"),
	})

	log.Printf("Product service listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *server) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "product name is required")
	}
	if req.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "price cannot be negative")
	}
	if req.StockQuantity < 0 {
		return nil, status.Error(codes.InvalidArgument, "stock quantity cannot be negative")
	}

	// Create product document
	product := bson.M{
		"name":           req.Name,
		"description":    req.Description,
		"price":         req.Price,
		"stock_quantity": req.StockQuantity,
		"category":      req.Category,
		"created_at":    time.Now().UTC(),
		"updated_at":    time.Now().UTC(),
	}

	// Insert into MongoDB
	result, err := s.db.Collection("products").InsertOne(ctx, product)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	// Get the inserted ID
	id := result.InsertedID.(primitive.ObjectID)

	// Return the created product
	return &pb.Product{
		Id:            id.Hex(),
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
		Category:      req.Category,
		CreatedAt:     product["created_at"].(time.Time).Format(time.RFC3339),
		UpdatedAt:     product["updated_at"].(time.Time).Format(time.RFC3339),
	}, nil
}

func (s *server) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}

	// Find product in MongoDB
	var product bson.M
	err = s.db.Collection("products").FindOne(ctx, bson.M{"_id": id}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}

	// Return the product
	return &pb.Product{
		Id:            req.Id,
		Name:          product["name"].(string),
		Description:   product["description"].(string),
		Price:         product["price"].(float64),
		StockQuantity: product["stock_quantity"].(int32),
		Category:      product["category"].(string),
		CreatedAt:     product["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		UpdatedAt:     product["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
	}, nil
}

func (s *server) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.Product, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}

	// Create update document
	update := bson.M{
		"$set": bson.M{
			"name":        req.Name,
			"description": req.Description,
			"price":       req.Price,
			"category":    req.Category,
			"updated_at":  time.Now().UTC(),
		},
	}

	// Find and update the product
	var updatedProduct bson.M
	err = s.db.Collection("products").FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedProduct)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update product: %v", err)
	}

	// Return the updated product
	return &pb.Product{
		Id:            req.Id,
		Name:          updatedProduct["name"].(string),
		Description:   updatedProduct["description"].(string),
		Price:         updatedProduct["price"].(float64),
		StockQuantity: updatedProduct["stock_quantity"].(int32),
		Category:      updatedProduct["category"].(string),
		CreatedAt:     updatedProduct["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		UpdatedAt:     updatedProduct["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
	}, nil
}

func (s *server) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.Product, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}

	// Create update document
	update := bson.M{
		"$inc": bson.M{
			"stock_quantity": req.QuantityChange,
		},
		"$set": bson.M{
			"updated_at": time.Now().UTC(),
		},
	}

	// Find and update the product
	var updatedProduct bson.M
	err = s.db.Collection("products").FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedProduct)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update product stock: %v", err)
	}

	// Check if stock went negative
	if updatedProduct["stock_quantity"].(int32) < 0 {
		// Revert the change
		_, err = s.db.Collection("products").UpdateOne(
			ctx,
			bson.M{"_id": id},
			bson.M{
				"$inc": bson.M{
					"stock_quantity": -req.QuantityChange,
				},
			},
		)
		return nil, status.Error(codes.FailedPrecondition, "insufficient stock")
	}

	// Return the updated product
	return &pb.Product{
		Id:            req.Id,
		Name:          updatedProduct["name"].(string),
		Description:   updatedProduct["description"].(string),
		Price:         updatedProduct["price"].(float64),
		StockQuantity: updatedProduct["stock_quantity"].(int32),
		Category:      updatedProduct["category"].(string),
		CreatedAt:     updatedProduct["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		UpdatedAt:     updatedProduct["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
	}, nil
}

func (s *server) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	// Set default values for pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 10
	}

	// Create filter
	filter := bson.M{}
	if req.Category != "" {
		filter["category"] = req.Category
	}

	// Calculate skip value for pagination
	skip := (req.Page - 1) * req.Limit

	// Get total count
	total, err := s.db.Collection("products").CountDocuments(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count products: %v", err)
	}

	// Find products
	cursor, err := s.db.Collection("products").Find(ctx, filter,
		options.Find().
			SetSkip(int64(skip)).
			SetLimit(int64(req.Limit)).
			SetSort(bson.M{"created_at": -1}),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list products: %v", err)
	}
	defer cursor.Close(ctx)

	// Process results
	products := make([]*pb.Product, 0)
	for cursor.Next(ctx) {
		var product bson.M
		if err := cursor.Decode(&product); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to decode product: %v", err)
		}

		products = append(products, &pb.Product{
			Id:            product["_id"].(primitive.ObjectID).Hex(),
			Name:          product["name"].(string),
			Description:   product["description"].(string),
			Price:         product["price"].(float64),
			StockQuantity: product["stock_quantity"].(int32),
			Category:      product["category"].(string),
			CreatedAt:     product["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
			UpdatedAt:     product["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		})
	}

	return &pb.ListProductsResponse{
		Products: products,
		Total:    int32(total),
	}, nil
}
