// Package database wires up connections to MongoDB and Redis.
package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"hms-backend/internal/config"
)

// ConnectMongo establishes a connection to MongoDB and returns both the
// client (for lifecycle management) and the configured database handle.
func ConnectMongo(ctx context.Context, cfg config.MongoConfig) (*mongo.Client, *mongo.Database, error) {
	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.URI)

	client, err := mongo.Connect(connectCtx, clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo: connect: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return nil, nil, fmt.Errorf("mongo: ping: %w", err)
	}

	return client, client.Database(cfg.Database), nil
}

// DisconnectMongo gracefully closes the MongoDB client connection.
func DisconnectMongo(ctx context.Context, client *mongo.Client) error {
	if client == nil {
		return nil
	}
	return client.Disconnect(ctx)
}
