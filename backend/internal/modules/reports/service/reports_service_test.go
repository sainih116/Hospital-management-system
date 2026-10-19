package service

import (
	"context"
	"errors"
	"testing"

	"hms-backend/internal/apperror"
	"hms-backend/internal/modules/reports/dto"
	"hms-backend/internal/modules/reports/repository"
)

func TestSummaryPassesThroughRepositoryData(t *testing.T) {
	repo := &fakeReportsRepository{summary: repository.SummaryData{
		TotalPatients: 42, TotalDoctors: 5, AppointmentsToday: 3,
		TotalRevenueCollected: 12345.67, OutstandingBalance: 890.10,
		LowStockMedicineCount: 2, PendingLabVerifications: 1, ActivePrescriptions: 7,
	}}
	svc := NewReportsService(Deps{Reports: repo})

	result, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary returned an unexpected error: %v", err)
	}
	if result.TotalPatients != 42 || result.TotalDoctors != 5 || result.AppointmentsToday != 3 {
		t.Errorf("expected counts to pass through unchanged, got %+v", result)
	}
	if result.TotalRevenueCollected != 12345.67 || result.OutstandingBalance != 890.10 {
		t.Errorf("expected money fields to pass through unchanged, got %+v", result)
	}
}

func TestDateRangeDefaultsToLast30Days(t *testing.T) {
	repo := &fakeReportsRepository{}
	svc := NewReportsService(Deps{Reports: repo})

	if _, err := svc.AppointmentsByStatus(context.Background(), dto.DateRangeQuery{}); err != nil {
		t.Fatalf("AppointmentsByStatus returned an unexpected error: %v", err)
	}

	gotDays := int(repo.lastTo.Sub(repo.lastFrom).Hours() / 24)
	if gotDays != defaultRangeDays {
		t.Errorf("expected the default range to span %d days, got %d (from=%v to=%v)", defaultRangeDays, gotDays, repo.lastFrom, repo.lastTo)
	}
}

func TestDateRangeParsesExplicitBounds(t *testing.T) {
	repo := &fakeReportsRepository{}
	svc := NewReportsService(Deps{Reports: repo})

	_, err := svc.RevenueTrend(context.Background(), dto.DateRangeQuery{From: "2026-01-01", To: "2026-01-31"})
	if err != nil {
		t.Fatalf("RevenueTrend returned an unexpected error: %v", err)
	}

	if repo.lastFrom.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("expected from=2026-01-01, got %v", repo.lastFrom)
	}
	if repo.lastTo.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("expected to=2026-01-31, got %v", repo.lastTo)
	}
	// `to` should be extended to the end of its day so same-day documents
	// still match.
	if repo.lastTo.Hour() != 23 || repo.lastTo.Minute() != 59 {
		t.Errorf("expected `to` to be extended to end-of-day, got %v", repo.lastTo)
	}
}

func TestDateRangeRejectsUnparsableDates(t *testing.T) {
	repo := &fakeReportsRepository{}
	svc := NewReportsService(Deps{Reports: repo})

	_, err := svc.PatientRegistrationTrend(context.Background(), dto.DateRangeQuery{From: "not-a-date"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrValidation.Code {
		t.Fatalf("expected a validation error for an unparsable from date, got: %v", err)
	}
}

func TestDateRangeRejectsFromAfterTo(t *testing.T) {
	repo := &fakeReportsRepository{}
	svc := NewReportsService(Deps{Reports: repo})

	_, err := svc.RevenueTrend(context.Background(), dto.DateRangeQuery{From: "2026-02-01", To: "2026-01-01"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrValidation.Code {
		t.Fatalf("expected a validation error when from is after to, got: %v", err)
	}
}

func TestDoctorWorkloadLimitDefaultsAndClamps(t *testing.T) {
	repo := &fakeReportsRepository{}
	svc := NewReportsService(Deps{Reports: repo})

	if _, err := svc.DoctorWorkload(context.Background(), dto.DoctorWorkloadQuery{}); err != nil {
		t.Fatalf("DoctorWorkload returned an unexpected error: %v", err)
	}
	if repo.lastLimit != defaultLimit {
		t.Errorf("expected the default limit %d when unset, got %d", defaultLimit, repo.lastLimit)
	}

	if _, err := svc.DoctorWorkload(context.Background(), dto.DoctorWorkloadQuery{Limit: 1000}); err != nil {
		t.Fatalf("DoctorWorkload returned an unexpected error: %v", err)
	}
	if repo.lastLimit != maxLimit {
		t.Errorf("expected the limit to clamp to %d, got %d", maxLimit, repo.lastLimit)
	}

	if _, err := svc.DoctorWorkload(context.Background(), dto.DoctorWorkloadQuery{Limit: 5}); err != nil {
		t.Fatalf("DoctorWorkload returned an unexpected error: %v", err)
	}
	if repo.lastLimit != 5 {
		t.Errorf("expected an explicit valid limit to pass through, got %d", repo.lastLimit)
	}
}

func TestLabTestVolumeMapsNameCounts(t *testing.T) {
	repo := &fakeReportsRepository{names: []repository.NameCount{
		{Name: "Complete Blood Count", Count: 12},
		{Name: "Lipid Profile", Count: 4},
	}}
	svc := NewReportsService(Deps{Reports: repo})

	result, err := svc.LabTestVolume(context.Background(), dto.DoctorWorkloadQuery{})
	if err != nil {
		t.Fatalf("LabTestVolume returned an unexpected error: %v", err)
	}
	if len(result) != 2 || result[0].Name != "Complete Blood Count" || result[0].Count != 12 {
		t.Errorf("expected mapped name counts, got %+v", result)
	}
}
