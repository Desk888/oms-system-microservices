#!/bin/bash

# Create proto directories in each service if they don't exist
mkdir -p order-service/proto
mkdir -p product-service/proto
mkdir -p user-service/proto
mkdir -p api-gateway/proto

# Copy all proto generated files to each service
cp proto/*.pb.go order-service/proto/
cp proto/*.pb.go product-service/proto/
cp proto/*.pb.go user-service/proto/
cp proto/*.pb.go api-gateway/proto/

echo "Proto files copied to all services"
