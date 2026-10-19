// Package sequence provides atomic, monotonically increasing counters
// backed by MongoDB, used to generate human-readable sequential codes
// (patient codes, lab sample codes, invoice numbers, ...) without relying
// on a relational database's native auto-increment.
package sequence

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store generates the next value for a named counter.
type Store struct {
	collection *mongo.Collection
}

// NewStore builds a MongoDB-backed sequence Store using the "counters"
// collection.
func NewStore(db *mongo.Database) *Store {
	return &Store{collection: db.Collection("counters")}
}

type counterDoc struct {
	Seq int64 `bson:"seq"`
}

// Next atomically increments and returns the counter named by key. The
// counter starts at 1 the first time it is used. findOneAndUpdate with
// upsert is atomic at the document level, so concurrent callers never
// receive the same value twice.
func (s *Store) Next(ctx context.Context, key string) (int64, error) {
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var doc counterDoc
	err := s.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": key},
		bson.M{"$inc": bson.M{"seq": 1}},
		opts,
	).Decode(&doc)
	if err != nil {
		return 0, fmt.Errorf("sequence: next(%s): %w", key, err)
	}

	return doc.Seq, nil
}
