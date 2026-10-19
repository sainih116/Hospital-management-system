package audit

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Log is the persisted audit_logs document.
type Log struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ActorID    string             `bson:"actorId,omitempty"`
	ActorEmail string             `bson:"actorEmail,omitempty"`
	ActorRole  string             `bson:"actorRole,omitempty"`
	Action     string             `bson:"action"`
	EntityType string             `bson:"entityType"`
	EntityID   string             `bson:"entityId,omitempty"`
	Metadata   map[string]any     `bson:"metadata,omitempty"`
	IP         string             `bson:"ip,omitempty"`
	CreatedAt  time.Time          `bson:"createdAt"`
}
