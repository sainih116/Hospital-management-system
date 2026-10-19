package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"

	"hms-backend/internal/apperror"
	"hms-backend/internal/modules/reports/dto"
)

// Export builds a multi-sheet .xlsx workbook — one sheet per report the
// dashboard already shows — for the same date range, reusing this
// service's own already-validated methods rather than re-querying the
// repository directly. A spreadsheet (not a server-rendered PDF) is the
// export format here for the same reason Lab Orders and Billing use
// browser print-to-PDF instead of a PDF library: it avoids a heavier,
// less-flexible dependency while still satisfying "exportable report"
// — recipients who want a PDF can print the resulting sheet, or print
// the dashboard itself via the browser.
func (s *reportsService) Export(ctx context.Context, query dto.DateRangeQuery) ([]byte, error) {
	summary, err := s.Summary(ctx)
	if err != nil {
		return nil, err
	}
	appointmentsByStatus, err := s.AppointmentsByStatus(ctx, query)
	if err != nil {
		return nil, err
	}
	revenueTrend, err := s.RevenueTrend(ctx, query)
	if err != nil {
		return nil, err
	}
	patientTrend, err := s.PatientRegistrationTrend(ctx, query)
	if err != nil {
		return nil, err
	}
	doctorWorkload, err := s.DoctorWorkload(ctx, dto.DoctorWorkloadQuery{From: query.From, To: query.To})
	if err != nil {
		return nil, err
	}
	labVolume, err := s.LabTestVolume(ctx, dto.DoctorWorkloadQuery{From: query.From, To: query.To})
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	defer f.Close()

	writeSummarySheet(f, summary)
	writeStatusCountSheet(f, "Appointments by Status", "Status", appointmentsByStatus)
	writeTrendSheet(f, "Revenue Trend", "Amount", revenueTrend)
	writeTrendSheet(f, "Patient Registrations", "Registrations", patientTrend)
	writeDoctorWorkloadSheet(f, doctorWorkload)
	writeLabVolumeSheet(f, labVolume)

	// excelize.NewFile() creates a default "Sheet1" — drop it now that
	// every real sheet above has been added, so it doesn't show up as a
	// confusing blank first tab.
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return nil, apperror.ErrInternal.WithCause(fmt.Errorf("reports: delete default sheet: %w", err))
	}
	f.SetActiveSheet(0)

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, apperror.ErrInternal.WithCause(fmt.Errorf("reports: write workbook: %w", err))
	}
	return buf.Bytes(), nil
}

func writeSummarySheet(f *excelize.File, s dto.SummaryResponse) {
	sheet := "Summary"
	_, _ = f.NewSheet(sheet)
	_ = f.SetSheetRow(sheet, "A1", &[]string{"Metric", "Value"})
	rows := [][]any{
		{"Total Patients", s.TotalPatients},
		{"Total Doctors", s.TotalDoctors},
		{"Appointments Today", s.AppointmentsToday},
		{"Total Revenue Collected", s.TotalRevenueCollected},
		{"Outstanding Balance", s.OutstandingBalance},
		{"Low Stock Medicines", s.LowStockMedicineCount},
		{"Pending Lab Verifications", s.PendingLabVerifications},
		{"Active Prescriptions", s.ActivePrescriptions},
	}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	_ = f.SetColWidth(sheet, "A", "A", 28)
}

func writeStatusCountSheet(f *excelize.File, sheet, labelHeader string, items []dto.StatusCountResponse) {
	_, _ = f.NewSheet(sheet)
	_ = f.SetSheetRow(sheet, "A1", &[]string{labelHeader, "Count"})
	for i, item := range items {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		row := []any{item.Status, item.Count}
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	_ = f.SetColWidth(sheet, "A", "A", 24)
}

func writeTrendSheet(f *excelize.File, sheet, valueHeader string, points []dto.TrendPointResponse) {
	_, _ = f.NewSheet(sheet)
	_ = f.SetSheetRow(sheet, "A1", &[]string{"Date", valueHeader})
	for i, p := range points {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		row := []any{p.Date, p.Value}
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	_ = f.SetColWidth(sheet, "A", "A", 16)
}

func writeDoctorWorkloadSheet(f *excelize.File, items []dto.DoctorWorkloadResponse) {
	sheet := "Doctor Workload"
	_, _ = f.NewSheet(sheet)
	_ = f.SetSheetRow(sheet, "A1", &[]string{"Doctor", "Appointments"})
	for i, item := range items {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		row := []any{item.DoctorName, item.Count}
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	_ = f.SetColWidth(sheet, "A", "A", 28)
}

func writeLabVolumeSheet(f *excelize.File, items []dto.NameCountResponse) {
	sheet := "Lab Test Volume"
	_, _ = f.NewSheet(sheet)
	_ = f.SetSheetRow(sheet, "A1", &[]string{"Test", "Orders"})
	for i, item := range items {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		row := []any{item.Name, item.Count}
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	_ = f.SetColWidth(sheet, "A", "A", 28)
}
