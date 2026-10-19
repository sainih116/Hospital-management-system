// Package repository implements MongoDB persistence for the doctors
// module.
package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hms-backend/internal/modules/doctors/domain"
	"hms-backend/internal/pkg/pagination"
)

// ErrNotFound is returned when a lookup finds no (non-deleted) document.
var ErrNotFound = errors.New("repository: not found")

// ErrLicenseTaken is returned when a license number is already registered.
var ErrLicenseTaken = errors.New("repository: license number already exists")

// ErrUserAlreadyLinked is returned when a user already has a doctor profile.
var ErrUserAlreadyLinked = errors.New("repository: user is already linked to a doctor profile")

// SearchFilter narrows a doctor search. Empty fields are not filtered on.
type SearchFilter struct {
	Query          string // matched against name, licenseNo and specialization
	DepartmentID   string
	Specialization string
}

// DoctorRepository persists and retrieves doctor profiles.
type DoctorRepository interface {
	Create(ctx context.Context, doctor *domain.Doctor) error
	FindByID(ctx context.Context, id string) (*domain.Doctor, error)
	FindByUserID(ctx context.Context, userID primitive.ObjectID) (*domain.Doctor, error)
	Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Doctor, error)
	SoftDelete(ctx context.Context, id primitive.ObjectID) error
	Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Doctor, int64, error)
}

type doctorRepository struct {
	collection *mongo.Collection
}

// NewDoctorRepository builds a MongoDB-backed DoctorRepository.
func NewDoctorRepository(db *mongo.Database) DoctorRepository {
	return &doctorRepository{collection: db.Collection("doctors")}
}

// EnsureIndexes creates the indexes the doctors collection relies on.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("doctors").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_userId"),
		},
		{
			Keys:    bson.D{{Key: "licenseNo", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_licenseNo"),
		},
		{
			Keys:    bson.D{{Key: "departmentId", Value: 1}},
			Options: options.Index().SetName("idx_departmentId"),
		},
		{
			Keys:    bson.D{{Key: "name", Value: "text"}, {Key: "specialization", Value: "text"}},
			Options: options.Index().SetName("txt_name_specialization"),
		},
		{
			Keys:    bson.D{{Key: "deletedAt", Value: 1}},
			Options: options.Index().SetName("idx_deletedAt"),
		},
	})
	return err
}

func notDeleted() bson.M {
	return bson.M{"deletedAt": bson.M{"$exists": false}}
}

func (r *doctorRepository) Create(ctx context.Context, doctor *domain.Doctor) error {
	now := time.Now().UTC()
	doctor.CreatedAt = now
	doctor.UpdatedAt = now
	if doctor.Status == "" {
		doctor.Status = domain.StatusActive
	}

	result, err := r.collection.InsertOne(ctx, doctor)
	if mongo.IsDuplicateKeyError(err) {
		// Determine which unique index tripped so the service can surface a
		// precise error; a targeted lookup is cheap and only happens on the
		// (rare) conflict path.
		if existing, findErr := r.FindByUserID(ctx, doctor.UserID); findErr == nil && existing != nil {
			return ErrUserAlreadyLinked
		}
		return ErrLicenseTaken
	}
	if err != nil {
		return err
	}
	doctor.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *doctorRepository) FindByID(ctx context.Context, id string) (*domain.Doctor, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}

	filter := notDeleted()
	filter["_id"] = oid

	var doctor domain.Doctor
	err = r.collection.FindOne(ctx, filter).Decode(&doctor)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &doctor, nil
}

func (r *doctorRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) (*domain.Doctor, error) {
	filter := notDeleted()
	filter["userId"] = userID

	var doctor domain.Doctor
	err := r.collection.FindOne(ctx, filter).Decode(&doctor)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &doctor, nil
}

func (r *doctorRepository) Update(ctx context.Context, id primitive.ObjectID, set bson.M) (*domain.Doctor, error) {
	set["updatedAt"] = time.Now().UTC()

	filter := notDeleted()
	filter["_id"] = id

	result := r.collection.FindOneAndUpdate(ctx, filter,
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var doctor domain.Doctor
	if err := result.Decode(&doctor); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doctor, nil
}

func (r *doctorRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	filter := notDeleted()
	filter["_id"] = id

	now := time.Now().UTC()
	result, err := r.collection.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{"deletedAt": now, "status": domain.StatusInactive, "updatedAt": now},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *doctorRepository) Search(ctx context.Context, filter SearchFilter, page pagination.Request) ([]domain.Doctor, int64, error) {
	query := notDeleted()

	if filter.DepartmentID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.DepartmentID); err == nil {
			query["departmentId"] = oid
		}
	}
	if filter.Specialization != "" {
		query["specialization"] = primitive.Regex{Pattern: regexp.QuoteMeta(filter.Specialization), Options: "i"}
	}
	if filter.Query != "" {
		pattern := primitive.Regex{Pattern: regexp.QuoteMeta(filter.Query), Options: "i"}
		query["$or"] = bson.A{
			bson.M{"name": pattern},
			bson.M{"licenseNo": pattern},
			bson.M{"specialization": pattern},
		}
	}

	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "name", Value: 1}}).
		SetSkip(page.Skip()).
		SetLimit(page.Limit())

	cursor, err := r.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var doctors []domain.Doctor
	if err := cursor.All(ctx, &doctors); err != nil {
		return nil, 0, err
	}

	return doctors, total, nil
}
