package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/xuri/excelize/v2"

	"hms-backend/internal/modules/reports/dto"
	"hms-backend/internal/modules/reports/repository"
)

func TestExportBuildsAWorkbookWithOneSheetPerReport(t *testing.T) {
	repo := &fakeReportsRepository{
		summary:  repository.SummaryData{TotalPatients: 5},
		statuses: []repository.StatusCount{{Status: "SCHEDULED", Count: 2}},
		trend:    []repository.TrendPoint{{Date: "2026-01-01", Value: 100}},
		workload: []repository.DoctorWorkload{{DoctorID: "d1", DoctorName: "Dr. Sharma", Count: 3}},
		names:    []repository.NameCount{{Name: "CBC", Count: 4}},
	}
	svc := NewReportsService(Deps{Reports: repo})

	fileBytes, err := svc.Export(context.Background(), dto.DateRangeQuery{})
	if err != nil {
		t.Fatalf("Export returned an unexpected error: %v", err)
	}
	if len(fileBytes) == 0 {
		t.Fatalf("expected a non-empty .xlsx file")
	}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatalf("expected the export to be a valid .xlsx file, failed to open: %v", err)
	}
	defer f.Close()

	wantSheets := []string{"Summary", "Appointments by Status", "Revenue Trend", "Patient Registrations", "Doctor Workload", "Lab Test Volume"}
	gotSheets := f.GetSheetList()
	for _, want := range wantSheets {
		found := false
		for _, got := range gotSheets {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected a %q sheet, got sheets: %v", want, gotSheets)
		}
	}

	if len(gotSheets) != len(wantSheets) {
		t.Errorf("expected exactly %d sheets (the default 'Sheet1' should be removed), got %d: %v", len(wantSheets), len(gotSheets), gotSheets)
	}

	rows, err := f.GetRows("Doctor Workload")
	if err != nil {
		t.Fatalf("failed to read Doctor Workload sheet: %v", err)
	}
	if len(rows) != 2 || rows[1][0] != "Dr. Sharma" {
		t.Errorf("expected the Doctor Workload sheet to contain the fake doctor, got %+v", rows)
	}
}
