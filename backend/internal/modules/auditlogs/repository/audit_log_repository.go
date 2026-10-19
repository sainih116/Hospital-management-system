// Package repository implements read-only MongoDB access to the
// audit_logs collection for the audit logging module's query/reporting
// API. Writes to this collection happen exclusively through
// internal/audit.Recorder, called by every other module as it performs
// auditable actions — this package never writes, only searches.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/audit"
	"hms-backend/internal/pkg/pagination"
)

// SearchFilter narrows an audit log search. Empty/zero fields are not
// filtered on.
type SearchFilter struct {
	ActorID    string
	EntityType string
	EntityID   string
	Action     string
	From       *time.Time
	To         *time.Time
}

// AuditLogRepository retrieves audit log entries.
type AuditLogRepository interface {
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]audit.Log, int64, error)
}

type auditLogRepository struct {
	collection *mongo.Collection
}

// NewAuditLogRepository builds a MongoDB-backed AuditLogRepository reading
// the same "audit_logs" collection internal/audit.MongoRecorder writes to.
func NewAuditLogRepository(db *mongo.Database) AuditLogRepository {
	return &auditLogRepository{collection: db.Collection("audit_logs")}
}

func (r *auditLogRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]audit.Log, int64, error) {
	query := bson.M{}

	if filter.ActorID != "" {
		query["actorId"] = filter.ActorID
	}
	if filter.EntityType != "" {
		query["entityType"] = filter.EntityType
	}
	if filter.EntityID != "" {
		query["entityId"] = filter.EntityID
	}
	if filter.Action != "" {
		query["action"] = filter.Action
	}
	if filter.From != nil || filter.To != nil {
		createdAt := bson.M{}
		if filter.From != nil {
			createdAt["$gte"] = *filter.From
		}
		if filter.To != nil {
			createdAt["$lte"] = *filter.To
		}
		query["createdAt"] = createdAt
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []audit.Log
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
