package audit

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRecorder persists audit entries to the audit_logs collection.
type MongoRecorder struct {
	collection *mongo.Collection
}

// NewMongoRecorder builds a MongoDB-backed Recorder.
func NewMongoRecorder(db *mongo.Database) *MongoRecorder {
	return &MongoRecorder{collection: db.Collection("audit_logs")}
}

// EnsureIndexes creates the indexes audit_logs relies on for efficient
// per-entity history lookups and per-actor activity queries.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("audit_logs").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "entityType", Value: 1}, {Key: "entityId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_entity_history"),
		},
		{
			Keys:    bson.D{{Key: "actorId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_actor_activity"),
		},
	})
	return err
}

// Record implements Recorder.
func (r *MongoRecorder) Record(ctx context.Context, entry Entry) error {
	log := Log{
		ActorID:    entry.ActorID,
		ActorEmail: entry.ActorEmail,
		ActorRole:  entry.ActorRole,
		Action:     entry.Action,
		EntityType: entry.EntityType,
		EntityID:   entry.EntityID,
		Metadata:   entry.Metadata,
		IP:         entry.IP,
		CreatedAt:  time.Now().UTC(),
	}

	_, err := r.collection.InsertOne(ctx, log)
	return err
}
