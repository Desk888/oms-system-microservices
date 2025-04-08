package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "github.com/order-management/proto"
)

type server struct {
	pb.UnimplementedUserServiceServer
	db *mongo.Database
}

type userModel struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Email        string            `bson:"email"`
	PasswordHash string            `bson:"password_hash"`
	FirstName    string            `bson:"first_name"`
	LastName     string            `bson:"last_name"`
	Role         string            `bson:"role"`
	Phone        string            `bson:"phone"`
	Address      string            `bson:"address"`
	CreatedAt    time.Time         `bson:"created_at"`
	UpdatedAt    time.Time         `bson:"updated_at"`
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

	// Create unique index for email
	_, err = client.Database("order_management").Collection("users").Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":50053") // Different port from other services
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{
		db: client.Database("order_management"),
	})

	log.Printf("User service listening on :50053")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	// Create user document
	user := userModel{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		Phone:        req.Phone,
		Address:      req.Address,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Insert into MongoDB
	result, err := s.db.Collection("users").InsertOne(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	// Get the inserted ID
	id := result.InsertedID.(primitive.ObjectID)

	// Return the created user (without password)
	return &pb.User{
		Id:        id.Hex(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
		Phone:     user.Phone,
		Address:   user.Address,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	// Find user in MongoDB
	var user userModel
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	// Return the user (without password)
	return &pb.User{
		Id:        req.Id,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
		Phone:     user.Phone,
		Address:   user.Address,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	// Create update document
	update := bson.M{
		"$set": bson.M{
			"email":      req.Email,
			"first_name": req.FirstName,
			"last_name":  req.LastName,
			"role":       req.Role,
			"phone":      req.Phone,
			"address":    req.Address,
			"updated_at": time.Now().UTC(),
		},
	}

	// Find and update the user
	var updatedUser userModel
	err = s.db.Collection("users").FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updatedUser)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	// Return the updated user (without password)
	return &pb.User{
		Id:        req.Id,
		Email:     updatedUser.Email,
		FirstName: updatedUser.FirstName,
		LastName:  updatedUser.LastName,
		Role:      updatedUser.Role,
		Phone:     updatedUser.Phone,
		Address:   updatedUser.Address,
		CreatedAt: updatedUser.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updatedUser.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *server) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	// Convert string ID to ObjectID
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	// Delete the user
	result, err := s.db.Collection("users").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	if result.DeletedCount == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &pb.DeleteUserResponse{
		Success: true,
		Message: "user deleted successfully",
	}, nil
}

func (s *server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	// Set default values for pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 10
	}

	// Create filter
	filter := bson.M{}
	if req.Role != "" {
		filter["role"] = req.Role
	}

	// Calculate skip value for pagination
	skip := (req.Page - 1) * req.Limit

	// Get total count
	total, err := s.db.Collection("users").CountDocuments(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count users: %v", err)
	}

	// Find users
	cursor, err := s.db.Collection("users").Find(ctx, filter,
		options.Find().
			SetSkip(int64(skip)).
			SetLimit(int64(req.Limit)).
			SetSort(bson.M{"created_at": -1}),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}
	defer cursor.Close(ctx)

	// Process results
	var users []*pb.User
	for cursor.Next(ctx) {
		var user userModel
		if err := cursor.Decode(&user); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to decode user: %v", err)
		}

		users = append(users, &pb.User{
			Id:        user.ID.Hex(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.Role,
			Phone:     user.Phone,
			Address:   user.Address,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		})
	}

	return &pb.ListUsersResponse{
		Users: users,
		Total: int32(total),
	}, nil
}

func (s *server) AuthenticateUser(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	// Find user by email
	var user userModel
	err := s.db.Collection("users").FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "invalid email or password")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	// Sign the token
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	return &pb.AuthResponse{
		Token: tokenString,
		User: &pb.User{
			Id:        user.ID.Hex(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.Role,
			Phone:     user.Phone,
			Address:   user.Address,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}
