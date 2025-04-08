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
	pb.UnimplementedOrderServiceServer
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
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterOrderServiceServer(s, &server{
		db: client.Database("order_management"),
	})

	log.Printf("Order service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "order must contain at least one item")
	}

	// Calculate total amount
	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += float64(item.Quantity) * item.Price
	}

	// Create order document
	order := bson.M{
		"user_id": req.UserId,
		"items":   req.Items,
		"status": "pending",
		"total_amount": totalAmount,
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}

	// Insert into MongoDB
	result, err := s.db.Collection("orders").InsertOne(ctx, order)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	// Get the inserted ID
	id := result.InsertedID.(primitive.ObjectID)

	// Return the created order
	return &pb.Order{
		Id:          id.Hex(),
		UserId:      req.UserId,
		Items:       req.Items,
		Status:      "pending",
		TotalAmount: totalAmount,
		CreatedAt:   order["created_at"].(time.Time).Format(time.RFC3339),
		UpdatedAt:   order["updated_at"].(time.Time).Format(time.RFC3339),
	}, nil
}

func (s *server) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}

	// Find order in MongoDB
	var order bson.M
	err = s.db.Collection("orders").FindOne(ctx, bson.M{"_id": id}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	// Convert items to proto format
	items := make([]*pb.OrderItem, 0)
	if itemsArray, ok := order["items"].(primitive.A); ok {
		for _, item := range itemsArray {
			if itemMap, ok := item.(primitive.M); ok {
				items = append(items, &pb.OrderItem{
					ProductId: itemMap["product_id"].(string),
					Quantity:  int32(itemMap["quantity"].(int64)),
					Price:     itemMap["price"].(float64),
				})
			}
		}
	}

	// Return the order
	return &pb.Order{
		Id:          req.Id,
		UserId:      order["user_id"].(string),
		Items:       items,
		Status:      order["status"].(string),
		TotalAmount: order["total_amount"].(float64),
		CreatedAt:   order["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		UpdatedAt:   order["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
	}, nil
}

func (s *server) UpdateOrder(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.Order, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}
	if req.Status == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}

	// Update order in MongoDB
	update := bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"updated_at": time.Now().UTC(),
		},
	}

	// Find and update the order
	var updatedOrder bson.M
	err = s.db.Collection("orders").FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedOrder)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update order: %v", err)
	}

	// Convert items to proto format
	items := make([]*pb.OrderItem, 0)
	if itemsArray, ok := updatedOrder["items"].(primitive.A); ok {
		for _, item := range itemsArray {
			if itemMap, ok := item.(primitive.M); ok {
				items = append(items, &pb.OrderItem{
					ProductId: itemMap["product_id"].(string),
					Quantity:  int32(itemMap["quantity"].(int64)),
					Price:     itemMap["price"].(float64),
				})
			}
		}
	}

	// Return the updated order
	return &pb.Order{
		Id:          req.Id,
		UserId:      updatedOrder["user_id"].(string),
		Items:       items,
		Status:      updatedOrder["status"].(string),
		TotalAmount: updatedOrder["total_amount"].(float64),
		CreatedAt:   updatedOrder["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		UpdatedAt:   updatedOrder["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
	}, nil
}

func (s *server) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	// Set default values for pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 10
	}

	// Create filter
	filter := bson.M{}
	if req.UserId != "" {
		filter["user_id"] = req.UserId
	}

	// Calculate skip value for pagination
	skip := (req.Page - 1) * req.Limit

	// Get total count
	total, err := s.db.Collection("orders").CountDocuments(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count orders: %v", err)
	}

	// Find orders
	cursor, err := s.db.Collection("orders").Find(ctx, filter,
		options.Find().
			SetSkip(int64(skip)).
			SetLimit(int64(req.Limit)).
			SetSort(bson.M{"created_at": -1}),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list orders: %v", err)
	}
	defer cursor.Close(ctx)

	// Process results
	orders := make([]*pb.Order, 0)
	for cursor.Next(ctx) {
		var order bson.M
		if err := cursor.Decode(&order); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to decode order: %v", err)
		}

		// Convert items to proto format
		items := make([]*pb.OrderItem, 0)
		if itemsArray, ok := order["items"].(primitive.A); ok {
			for _, item := range itemsArray {
				if itemMap, ok := item.(primitive.M); ok {
					items = append(items, &pb.OrderItem{
						ProductId: itemMap["product_id"].(string),
						Quantity:  int32(itemMap["quantity"].(int64)),
						Price:     itemMap["price"].(float64),
					})
				}
			}
		}

		orders = append(orders, &pb.Order{
			Id:          order["_id"].(primitive.ObjectID).Hex(),
			UserId:      order["user_id"].(string),
			Items:       items,
			Status:      order["status"].(string),
			TotalAmount: order["total_amount"].(float64),
			CreatedAt:   order["created_at"].(primitive.DateTime).Time().Format(time.RFC3339),
			UpdatedAt:   order["updated_at"].(primitive.DateTime).Time().Format(time.RFC3339),
		})
	}

	return &pb.ListOrdersResponse{
		Orders: orders,
		Total:  int32(total),
	}, nil
}
