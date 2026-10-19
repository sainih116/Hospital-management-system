// Package repository implements the reporting module's read-only
// aggregation queries. Unlike every other module, this one intentionally
// reaches directly into sibling modules' collections (patients, doctors,
// appointments, bills, payments, medicines, lab_orders, prescriptions)
// instead of going through their repository interfaces — a dashboard
// needs cross-cutting Mongo aggregation pipelines ($group/$sum/$dateToString),
// not the single-collection CRUD/search each module's own repository
// exposes. Nothing here ever writes.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SummaryData is the dashboard's top-level snapshot: one cheap count/sum
// per module, computed independently (not from the trend endpoints) so
// the summary cards load fast without pulling any date range.
type SummaryData struct {
	TotalPatients           int64
	TotalDoctors            int64
	AppointmentsToday       int64
	TotalRevenueCollected   float64
	OutstandingBalance      float64
	LowStockMedicineCount   int64
	PendingLabVerifications int64
	ActivePrescriptions     int64
}

// StatusCount is one (status, count) bucket.
type StatusCount struct {
	Status string
	Count  int64
}

// TrendPoint is one (date, value) point on a time-series chart. Date is
// formatted "YYYY-MM-DD" (UTC) by the aggregation pipeline itself, so the
// frontend can render it directly without re-parsing a timestamp.
type TrendPoint struct {
	Date  string
	Value float64
}

// DoctorWorkload is one doctor's appointment count within a date range.
type DoctorWorkload struct {
	DoctorID   string
	DoctorName string
	Count      int64
}

// NameCount is one (name, count) bucket — used for lab test volume.
type NameCount struct {
	Name  string
	Count int64
}

// ReportsRepository runs the aggregation queries behind the reporting
// dashboard.
type ReportsRepository interface {
	Summary(ctx context.Context) (SummaryData, error)
	AppointmentsByStatus(ctx context.Context, from, to time.Time) ([]StatusCount, error)
	RevenueTrend(ctx context.Context, from, to time.Time) ([]TrendPoint, error)
	PatientRegistrationTrend(ctx context.Context, from, to time.Time) ([]TrendPoint, error)
	DoctorWorkload(ctx context.Context, from, to time.Time, limit int) ([]DoctorWorkload, error)
	LabTestVolume(ctx context.Context, from, to time.Time, limit int) ([]NameCount, error)
}

type reportsRepository struct {
	patients      *mongo.Collection
	doctors       *mongo.Collection
	appointments  *mongo.Collection
	bills         *mongo.Collection
	payments      *mongo.Collection
	medicines     *mongo.Collection
	labOrders     *mongo.Collection
	prescriptions *mongo.Collection
}

// NewReportsRepository builds a MongoDB-backed ReportsRepository.
func NewReportsRepository(db *mongo.Database) ReportsRepository {
	return &reportsRepository{
		patients:      db.Collection("patients"),
		doctors:       db.Collection("doctors"),
		appointments:  db.Collection("appointments"),
		bills:         db.Collection("bills"),
		payments:      db.Collection("payments"),
		medicines:     db.Collection("medicines"),
		labOrders:     db.Collection("lab_orders"),
		prescriptions: db.Collection("prescriptions"),
	}
}

func (r *reportsRepository) Summary(ctx context.Context) (SummaryData, error) {
	var data SummaryData

	totalPatients, err := r.patients.CountDocuments(ctx, bson.M{"deletedAt": bson.M{"$exists": false}})
	if err != nil {
		return data, err
	}
	data.TotalPatients = totalPatients

	totalDoctors, err := r.doctors.CountDocuments(ctx, bson.M{"status": "ACTIVE"})
	if err != nil {
		return data, err
	}
	data.TotalDoctors = totalDoctors

	todayStart := time.Now().UTC().Truncate(24 * time.Hour)
	todayEnd := todayStart.Add(24 * time.Hour)
	appointmentsToday, err := r.appointments.CountDocuments(ctx, bson.M{"startAt": bson.M{"$gte": todayStart, "$lt": todayEnd}})
	if err != nil {
		return data, err
	}
	data.AppointmentsToday = appointmentsToday

	revenue, err := sumField(ctx, r.payments, bson.M{}, "amount")
	if err != nil {
		return data, err
	}
	data.TotalRevenueCollected = revenue

	outstanding, err := sumExpr(ctx, r.bills,
		bson.M{"status": bson.M{"$in": bson.A{"UNPAID", "PARTIALLY_PAID"}}},
		bson.M{"$subtract": bson.A{"$totalAmount", "$amountPaid"}},
	)
	if err != nil {
		return data, err
	}
	data.OutstandingBalance = outstanding

	lowStock, err := r.medicines.CountDocuments(ctx, bson.M{
		"status": "ACTIVE",
		"$expr":  bson.M{"$lte": bson.A{"$currentStock", "$reorderLevel"}},
	})
	if err != nil {
		return data, err
	}
	data.LowStockMedicineCount = lowStock

	pendingLab, err := r.labOrders.CountDocuments(ctx, bson.M{"status": "AWAITING_VERIFICATION"})
	if err != nil {
		return data, err
	}
	data.PendingLabVerifications = pendingLab

	activePrescriptions, err := r.prescriptions.CountDocuments(ctx, bson.M{"status": "ACTIVE"})
	if err != nil {
		return data, err
	}
	data.ActivePrescriptions = activePrescriptions

	return data, nil
}

func (r *reportsRepository) AppointmentsByStatus(ctx context.Context, from, to time.Time) ([]StatusCount, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"startAt": bson.M{"$gte": from, "$lte": to}}},
		bson.M{"$group": bson.M{"_id": "$status", "count": bson.M{"$sum": 1}}},
		bson.M{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := r.appointments.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	results := make([]StatusCount, 0, len(docs))
	for _, d := range docs {
		results = append(results, StatusCount{Status: d.ID, Count: d.Count})
	}
	return results, nil
}

func (r *reportsRepository) RevenueTrend(ctx context.Context, from, to time.Time) ([]TrendPoint, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"paidAt": bson.M{"$gte": from, "$lte": to}}},
		bson.M{"$group": bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$paidAt"}},
			"total": bson.M{"$sum": "$amount"},
		}},
		bson.M{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := r.payments.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []struct {
		ID    string  `bson:"_id"`
		Total float64 `bson:"total"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	results := make([]TrendPoint, 0, len(docs))
	for _, d := range docs {
		results = append(results, TrendPoint{Date: d.ID, Value: d.Total})
	}
	return results, nil
}

func (r *reportsRepository) PatientRegistrationTrend(ctx context.Context, from, to time.Time) ([]TrendPoint, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"createdAt": bson.M{"$gte": from, "$lte": to}}},
		bson.M{"$group": bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"}},
			"count": bson.M{"$sum": 1},
		}},
		bson.M{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := r.patients.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	results := make([]TrendPoint, 0, len(docs))
	for _, d := range docs {
		results = append(results, TrendPoint{Date: d.ID, Value: float64(d.Count)})
	}
	return results, nil
}

func (r *reportsRepository) DoctorWorkload(ctx context.Context, from, to time.Time, limit int) ([]DoctorWorkload, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"startAt": bson.M{"$gte": from, "$lte": to}}},
		bson.M{"$group": bson.M{
			"_id":        "$doctorId",
			"doctorName": bson.M{"$first": "$doctorName"},
			"count":      bson.M{"$sum": 1},
		}},
		bson.M{"$sort": bson.M{"count": -1}},
		bson.M{"$limit": limit},
	}

	cursor, err := r.appointments.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []struct {
		ID         primitive.ObjectID `bson:"_id"`
		DoctorName string             `bson:"doctorName"`
		Count      int64              `bson:"count"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	results := make([]DoctorWorkload, 0, len(docs))
	for _, d := range docs {
		results = append(results, DoctorWorkload{DoctorID: d.ID.Hex(), DoctorName: d.DoctorName, Count: d.Count})
	}
	return results, nil
}

func (r *reportsRepository) LabTestVolume(ctx context.Context, from, to time.Time, limit int) ([]NameCount, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"createdAt": bson.M{"$gte": from, "$lte": to}}},
		bson.M{"$group": bson.M{"_id": "$testName", "count": bson.M{"$sum": 1}}},
		bson.M{"$sort": bson.M{"count": -1}},
		bson.M{"$limit": limit},
	}

	cursor, err := r.labOrders.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	results := make([]NameCount, 0, len(docs))
	for _, d := range docs {
		results = append(results, NameCount{Name: d.ID, Count: d.Count})
	}
	return results, nil
}

// sumField returns the sum of a numeric field across every document
// matching filter, or 0 if none match.
func sumField(ctx context.Context, collection *mongo.Collection, filter bson.M, field string) (float64, error) {
	return sumExpr(ctx, collection, filter, "$"+field)
}

// sumExpr returns the sum of an arbitrary aggregation expression (a field
// reference or a computed $subtract/$multiply/... expression) across every
// document matching filter, or 0 if none match.
func sumExpr(ctx context.Context, collection *mongo.Collection, filter bson.M, expr any) (float64, error) {
	pipeline := bson.A{
		bson.M{"$match": filter},
		bson.M{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": expr}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var docs []struct {
		Total float64 `bson:"total"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return 0, err
	}
	if len(docs) == 0 {
		return 0, nil
	}
	return docs[0].Total, nil
}
