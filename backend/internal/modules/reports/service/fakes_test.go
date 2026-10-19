package service

import (
	"context"
	"time"

	"hms-backend/internal/modules/reports/repository"
)

// fakeReportsRepository is a minimal in-memory implementation of
// repository.ReportsRepository that records the arguments each method was
// called with (so tests can assert the service parsed/defaulted a
// date-range or limit correctly) and returns whatever canned data the
// test pre-loads.
type fakeReportsRepository struct {
	summary  repository.SummaryData
	statuses []repository.StatusCount
	trend    []repository.TrendPoint
	workload []repository.DoctorWorkload
	names    []repository.NameCount

	lastFrom  time.Time
	lastTo    time.Time
	lastLimit int
}

func (f *fakeReportsRepository) Summary(context.Context) (repository.SummaryData, error) {
	return f.summary, nil
}

func (f *fakeReportsRepository) AppointmentsByStatus(_ context.Context, from, to time.Time) ([]repository.StatusCount, error) {
	f.lastFrom, f.lastTo = from, to
	return f.statuses, nil
}

func (f *fakeReportsRepository) RevenueTrend(_ context.Context, from, to time.Time) ([]repository.TrendPoint, error) {
	f.lastFrom, f.lastTo = from, to
	return f.trend, nil
}

func (f *fakeReportsRepository) PatientRegistrationTrend(_ context.Context, from, to time.Time) ([]repository.TrendPoint, error) {
	f.lastFrom, f.lastTo = from, to
	return f.trend, nil
}

func (f *fakeReportsRepository) DoctorWorkload(_ context.Context, from, to time.Time, limit int) ([]repository.DoctorWorkload, error) {
	f.lastFrom, f.lastTo, f.lastLimit = from, to, limit
	return f.workload, nil
}

func (f *fakeReportsRepository) LabTestVolume(_ context.Context, from, to time.Time, limit int) ([]repository.NameCount, error) {
	f.lastFrom, f.lastTo, f.lastLimit = from, to, limit
	return f.names, nil
}
