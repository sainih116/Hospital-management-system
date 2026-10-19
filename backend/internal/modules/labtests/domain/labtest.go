// Package domain holds the lab tests module's core entity — the test
// catalog (CBC, HbA1c, LFT, ...) that lab orders are placed against.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status tracks whether a test is currently orderable. Never hard-deleted
// so historical lab orders always resolve to a valid test.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// LabTest is one catalog entry: what can be ordered, its sample
// requirements, reference range and cost.
type LabTest struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Code        string             `bson:"code"`
	Name        string             `bson:"name"`
	Category    string             `bson:"category,omitempty"`
	SampleType  string             `bson:"sampleType,omitempty"`
	Unit        string             `bson:"unit,omitempty"`
	NormalRange string             `bson:"normalRange,omitempty"`
	Cost        float64            `bson:"cost"`
	Status      Status             `bson:"status"`
	CreatedAt   time.Time          `bson:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"`
}
