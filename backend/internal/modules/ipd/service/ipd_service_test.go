package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	patientsDomain "hms-backend/internal/modules/patients/domain"

	"hms-backend/internal/modules/ipd/domain"
	"hms-backend/internal/modules/ipd/dto"
)

type testFixture struct {
	wards      *fakeWardRepository
	beds       *fakeBedRepository
	admissions *fakeAdmissionRepository
	patients   *fakePatientRepository
	doctors    *fakeDoctorRepository
	audit      *fakeAuditRecorder
	svc        IPDService
}

func newTestFixture() *testFixture {
	f := &testFixture{
		wards: newFakeWardRepository(), beds: newFakeBedRepository(), admissions: newFakeAdmissionRepository(),
		patients: newFakePatientRepository(), doctors: newFakeDoctorRepository(), audit: &fakeAuditRecorder{},
	}
	f.svc = NewIPDService(Deps{
		Wards: f.wards, Beds: f.beds, Admissions: f.admissions,
		Patients: f.patients, Doctors: f.doctors, Audit: f.audit, Logger: zap.NewNop(),
	})
	return f
}

func testActor() audit.Actor {
	return audit.Actor{UserID: "", Email: "nurse@hms.local", Role: "NURSE"}
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func TestCreateWard(t *testing.T) {
	f := newTestFixture()

	ward, err := f.svc.CreateWard(context.Background(), dto.CreateWardRequest{Name: "ICU", Code: "ICU"}, testActor())
	if err != nil {
		t.Fatalf("CreateWard returned an unexpected error: %v", err)
	}
	if ward.Name != "ICU" || ward.Status != "ACTIVE" {
		t.Errorf("unexpected ward: %+v", ward)
	}
}

func TestCreateBedRequiresAnExistingWard(t *testing.T) {
	f := newTestFixture()

	_, err := f.svc.CreateBed(context.Background(), dto.CreateBedRequest{WardID: "000000000000000000000000", BedNumber: "B1"}, testActor())
	if err == nil {
		t.Fatalf("expected an error for a nonexistent ward")
	}
	if code := appErrCode(t, err); code != apperror.ErrValidation.Code {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestAdmitPatientOccupiesTheBedAndSnapshotsNames(t *testing.T) {
	f := newTestFixture()

	ward := f.wards.seed(&domain.Ward{Name: "General Ward", Code: "GEN"})
	bed := f.beds.seed(&domain.Bed{WardID: ward.ID, WardName: ward.Name, BedNumber: "GEN-01"})
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Rohan Kumar"})
	doctor := f.doctors.seed(&doctorsDomain.Doctor{Name: "Dr. Anil Sharma"})

	admission, err := f.svc.AdmitPatient(context.Background(), dto.AdmitPatientRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), BedID: bed.ID.Hex(), Reason: "Observation",
	}, testActor())
	if err != nil {
		t.Fatalf("AdmitPatient returned an unexpected error: %v", err)
	}
	if admission.PatientName != "Rohan Kumar" || admission.DoctorName != "Dr. Anil Sharma" || admission.BedNumber != "GEN-01" {
		t.Errorf("expected snapshotted names, got %+v", admission)
	}
	if admission.Status != "ADMITTED" {
		t.Errorf("expected status ADMITTED, got %s", admission.Status)
	}

	updatedBed, _ := f.beds.FindByID(context.Background(), bed.ID.Hex())
	if updatedBed.Status != domain.BedOccupied {
		t.Errorf("expected the bed to be OCCUPIED after admission, got %s", updatedBed.Status)
	}

	actions := f.audit.actions()
	if len(actions) != 1 || actions[0] != "PATIENT_ADMITTED" {
		t.Errorf("expected a PATIENT_ADMITTED audit entry, got %v", actions)
	}
}

// TestAdmitPatientRejectsAnAlreadyOccupiedBed is the blueprint's explicit
// critical test case: "a bed cannot be assigned to two active admissions
// at the same time." The atomic Occupy guard-filter (not an application-
// level check-then-act) is what actually closes this race in production;
// this test proves the *outcome* the guard is there to guarantee.
func TestAdmitPatientRejectsAnAlreadyOccupiedBed(t *testing.T) {
	f := newTestFixture()

	ward := f.wards.seed(&domain.Ward{Name: "ICU", Code: "ICU"})
	bed := f.beds.seed(&domain.Bed{WardID: ward.ID, WardName: ward.Name, BedNumber: "ICU-01", Status: domain.BedOccupied})
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Neha Singh"})
	doctor := f.doctors.seed(&doctorsDomain.Doctor{Name: "Dr. Neha Verma"})

	_, err := f.svc.AdmitPatient(context.Background(), dto.AdmitPatientRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), BedID: bed.ID.Hex(),
	}, testActor())
	if err == nil {
		t.Fatalf("expected an error admitting into an already-occupied bed")
	}
	if code := appErrCode(t, err); code != "BED_NOT_AVAILABLE" {
		t.Fatalf("expected BED_NOT_AVAILABLE, got %s", code)
	}
}

func TestDischargePatientReleasesTheBed(t *testing.T) {
	f := newTestFixture()

	ward := f.wards.seed(&domain.Ward{Name: "General Ward", Code: "GEN"})
	bed := f.beds.seed(&domain.Bed{WardID: ward.ID, WardName: ward.Name, BedNumber: "GEN-01"})
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Arjun Joshi"})
	doctor := f.doctors.seed(&doctorsDomain.Doctor{Name: "Dr. Rahul Mehta"})

	admission, err := f.svc.AdmitPatient(context.Background(), dto.AdmitPatientRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), BedID: bed.ID.Hex(),
	}, testActor())
	if err != nil {
		t.Fatalf("AdmitPatient returned an unexpected error: %v", err)
	}

	discharged, err := f.svc.DischargePatient(context.Background(), admission.ID, dto.DischargeRequest{Note: "Recovered"}, testActor())
	if err != nil {
		t.Fatalf("DischargePatient returned an unexpected error: %v", err)
	}
	if discharged.Status != "DISCHARGED" || discharged.DischargedAt == nil {
		t.Errorf("expected a discharged admission with a timestamp, got %+v", discharged)
	}

	updatedBed, _ := f.beds.FindByID(context.Background(), bed.ID.Hex())
	if updatedBed.Status != domain.BedAvailable {
		t.Errorf("expected the bed to be released back to AVAILABLE, got %s", updatedBed.Status)
	}

	// The now-available bed should accept a brand new admission.
	patient2 := f.patients.seed(&patientsDomain.Patient{Name: "Second Patient"})
	if _, err := f.svc.AdmitPatient(context.Background(), dto.AdmitPatientRequest{
		PatientID: patient2.ID.Hex(), DoctorID: doctor.ID.Hex(), BedID: bed.ID.Hex(),
	}, testActor()); err != nil {
		t.Fatalf("expected the released bed to accept a new admission, got: %v", err)
	}
}

func TestDischargePatientRejectsANonActiveAdmission(t *testing.T) {
	f := newTestFixture()

	ward := f.wards.seed(&domain.Ward{Name: "General Ward", Code: "GEN"})
	bed := f.beds.seed(&domain.Bed{WardID: ward.ID, WardName: ward.Name, BedNumber: "GEN-01"})
	patient := f.patients.seed(&patientsDomain.Patient{Name: "Rohan Kumar"})
	doctor := f.doctors.seed(&doctorsDomain.Doctor{Name: "Dr. Anil Sharma"})

	admission, _ := f.svc.AdmitPatient(context.Background(), dto.AdmitPatientRequest{
		PatientID: patient.ID.Hex(), DoctorID: doctor.ID.Hex(), BedID: bed.ID.Hex(),
	}, testActor())
	if _, err := f.svc.DischargePatient(context.Background(), admission.ID, dto.DischargeRequest{}, testActor()); err != nil {
		t.Fatalf("first discharge returned an unexpected error: %v", err)
	}

	_, err := f.svc.DischargePatient(context.Background(), admission.ID, dto.DischargeRequest{}, testActor())
	if err == nil {
		t.Fatalf("expected an error discharging an already-discharged admission")
	}
	if code := appErrCode(t, err); code != "ADMISSION_NOT_ACTIVE" {
		t.Fatalf("expected ADMISSION_NOT_ACTIVE, got %s", code)
	}
}
