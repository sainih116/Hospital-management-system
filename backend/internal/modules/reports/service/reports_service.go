// Package service implements the reporting module's business logic:
// parsing/defaulting date-range filters and mapping aggregation results
// to their public representation. Every metric is computed live from the
// operational collections (no pre-aggregated/materialized report tables),
// which keeps the dashboard always current at the cost of running a
// handful of aggregation pipelines per request — an acceptable tradeoff
// at this system's scale, and consistent with every other module's
// "correct and simple over premature optimization" bias.
package service

import (
	"context"
	"time"

	"hms-backend/internal/apperror"

	"hms-backend/internal/modules/reports/dto"
	"hms-backend/internal/modules/reports/repository"
)

const (
	defaultRangeDays = 30
	defaultLimit     = 10
	maxLimit         = 50
	dateLayout       = "2006-01-02"
)

// ReportsService is the reporting module's public business-logic
// contract.
type ReportsService interface {
	Summary(ctx context.Context) (dto.SummaryResponse, error)
	AppointmentsByStatus(ctx context.Context, query dto.DateRangeQuery) ([]dto.StatusCountResponse, error)
	RevenueTrend(ctx context.Context, query dto.DateRangeQuery) ([]dto.TrendPointResponse, error)
	PatientRegistrationTrend(ctx context.Context, query dto.DateRangeQuery) ([]dto.TrendPointResponse, error)
	DoctorWorkload(ctx context.Context, query dto.DoctorWorkloadQuery) ([]dto.DoctorWorkloadResponse, error)
	LabTestVolume(ctx context.Context, query dto.DoctorWorkloadQuery) ([]dto.NameCountResponse, error)
	// Export builds a multi-sheet .xlsx workbook covering every metric
	// above for the same date range — see export_service.go.
	Export(ctx context.Context, query dto.DateRangeQuery) ([]byte, error)
}

type reportsService struct {
	reports repository.ReportsRepository
}

// Deps bundles every collaborator reportsService needs.
type Deps struct {
	Reports repository.ReportsRepository
}

// NewReportsService builds the reporting module's service implementation.
func NewReportsService(d Deps) ReportsService {
	return &reportsService{reports: d.Reports}
}

func (s *reportsService) Summary(ctx context.Context) (dto.SummaryResponse, error) {
	data, err := s.reports.Summary(ctx)
	if err != nil {
		return dto.SummaryResponse{}, apperror.ErrInternal.WithCause(err)
	}

	return dto.SummaryResponse{
		TotalPatients: data.TotalPatients, TotalDoctors: data.TotalDoctors,
		AppointmentsToday: data.AppointmentsToday, TotalRevenueCollected: data.TotalRevenueCollected,
		OutstandingBalance: data.OutstandingBalance, LowStockMedicineCount: data.LowStockMedicineCount,
		PendingLabVerifications: data.PendingLabVerifications, ActivePrescriptions: data.ActivePrescriptions,
	}, nil
}

func (s *reportsService) AppointmentsByStatus(ctx context.Context, query dto.DateRangeQuery) ([]dto.StatusCountResponse, error) {
	from, to, err := parseRange(query.From, query.To)
	if err != nil {
		return nil, err
	}

	counts, err := s.reports.AppointmentsByStatus(ctx, from, to)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.StatusCountResponse, 0, len(counts))
	for _, c := range counts {
		items = append(items, dto.StatusCountResponse{Status: c.Status, Count: c.Count})
	}
	return items, nil
}

func (s *reportsService) RevenueTrend(ctx context.Context, query dto.DateRangeQuery) ([]dto.TrendPointResponse, error) {
	from, to, err := parseRange(query.From, query.To)
	if err != nil {
		return nil, err
	}

	points, err := s.reports.RevenueTrend(ctx, from, to)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}
	return toTrendResponse(points), nil
}

func (s *reportsService) PatientRegistrationTrend(ctx context.Context, query dto.DateRangeQuery) ([]dto.TrendPointResponse, error) {
	from, to, err := parseRange(query.From, query.To)
	if err != nil {
		return nil, err
	}

	points, err := s.reports.PatientRegistrationTrend(ctx, from, to)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}
	return toTrendResponse(points), nil
}

func (s *reportsService) DoctorWorkload(ctx context.Context, query dto.DoctorWorkloadQuery) ([]dto.DoctorWorkloadResponse, error) {
	from, to, err := parseRange(query.From, query.To)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(query.Limit)

	workload, err := s.reports.DoctorWorkload(ctx, from, to, limit)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.DoctorWorkloadResponse, 0, len(workload))
	for _, w := range workload {
		items = append(items, dto.DoctorWorkloadResponse{DoctorID: w.DoctorID, DoctorName: w.DoctorName, Count: w.Count})
	}
	return items, nil
}

func (s *reportsService) LabTestVolume(ctx context.Context, query dto.DoctorWorkloadQuery) ([]dto.NameCountResponse, error) {
	from, to, err := parseRange(query.From, query.To)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(query.Limit)

	volume, err := s.reports.LabTestVolume(ctx, from, to, limit)
	if err != nil {
		return nil, apperror.ErrInternal.WithCause(err)
	}

	items := make([]dto.NameCountResponse, 0, len(volume))
	for _, v := range volume {
		items = append(items, dto.NameCountResponse{Name: v.Name, Count: v.Count})
	}
	return items, nil
}

// parseRange parses "YYYY-MM-DD" from/to bounds, defaulting to the last
// defaultRangeDays days (inclusive of today) when either is omitted. `to`
// is extended to the end of its day so a same-day range still matches
// documents created any time on that day.
func parseRange(fromStr, toStr string) (time.Time, time.Time, error) {
	now := time.Now().UTC()

	to := now
	if toStr != "" {
		parsed, err := time.Parse(dateLayout, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, apperror.ErrValidation.WithDetails(map[string]string{"to": "must be formatted YYYY-MM-DD"})
		}
		to = parsed
	}
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)

	from := to.AddDate(0, 0, -defaultRangeDays)
	if fromStr != "" {
		parsed, err := time.Parse(dateLayout, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, apperror.ErrValidation.WithDetails(map[string]string{"from": "must be formatted YYYY-MM-DD"})
		}
		from = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	}

	if from.After(to) {
		return time.Time{}, time.Time{}, apperror.ErrValidation.WithDetails(map[string]string{"from": "must not be after to"})
	}

	return from, to, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func toTrendResponse(points []repository.TrendPoint) []dto.TrendPointResponse {
	items := make([]dto.TrendPointResponse, 0, len(points))
	for _, p := range points {
		items = append(items, dto.TrendPointResponse{Date: p.Date, Value: p.Value})
	}
	return items
}
