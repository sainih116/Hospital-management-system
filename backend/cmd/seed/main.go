// Command seed bootstraps a fresh database: it upserts the six RBAC roles
// (from domain.DefaultRolePermissions) and, if no user exists yet with the
// configured admin email, creates the initial Administrator account. It is
// idempotent — safe to run on every deployment, not just the first one.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"hms-backend/internal/audit"
	"hms-backend/internal/config"
	"hms-backend/internal/database"
	appointmentsRepo "hms-backend/internal/modules/appointments/repository"
	"hms-backend/internal/modules/auth/domain"
	"hms-backend/internal/modules/auth/repository"
	billingRepo "hms-backend/internal/modules/billing/repository"
	departmentsDomain "hms-backend/internal/modules/departments/domain"
	departmentsRepo "hms-backend/internal/modules/departments/repository"
	doctorsDomain "hms-backend/internal/modules/doctors/domain"
	doctorsRepo "hms-backend/internal/modules/doctors/repository"
	ipdDomain "hms-backend/internal/modules/ipd/domain"
	ipdRepo "hms-backend/internal/modules/ipd/repository"
	laborders "hms-backend/internal/modules/laborders/repository"
	labtestsDomain "hms-backend/internal/modules/labtests/domain"
	labtestsRepo "hms-backend/internal/modules/labtests/repository"
	medicalRecordsRepo "hms-backend/internal/modules/medicalrecords/repository"
	patientsRepo "hms-backend/internal/modules/patients/repository"
	pharmacyDomain "hms-backend/internal/modules/pharmacy/domain"
	pharmacyRepo "hms-backend/internal/modules/pharmacy/repository"
	prescriptionsRepo "hms-backend/internal/modules/prescriptions/repository"
	"hms-backend/internal/security"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, db, err := database.ConnectMongo(ctx, cfg.Mongo)
	if err != nil {
		return fmt.Errorf("connect mongo: %w", err)
	}
	defer func() { _ = database.DisconnectMongo(context.Background(), client) }()

	if err := repository.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure auth indexes: %w", err)
	}
	if err := patientsRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure patients indexes: %w", err)
	}
	if err := departmentsRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure departments indexes: %w", err)
	}
	if err := doctorsRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure doctors indexes: %w", err)
	}
	if err := appointmentsRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure appointments indexes: %w", err)
	}
	if err := medicalRecordsRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure medical records indexes: %w", err)
	}
	if err := prescriptionsRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure prescriptions indexes: %w", err)
	}
	if err := labtestsRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure lab tests indexes: %w", err)
	}
	if err := laborders.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure lab orders indexes: %w", err)
	}
	if err := pharmacyRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure pharmacy indexes: %w", err)
	}
	if err := billingRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure billing indexes: %w", err)
	}
	if err := audit.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure audit indexes: %w", err)
	}
	if err := ipdRepo.EnsureIndexes(ctx, db); err != nil {
		return fmt.Errorf("ensure IPD indexes: %w", err)
	}

	if err := seedRoles(ctx, db); err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}

	if err := seedAdminUser(ctx, db); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	departmentsByCode, err := seedDepartments(ctx, db)
	if err != nil {
		return fmt.Errorf("seed departments: %w", err)
	}

	if err := seedSampleDoctors(ctx, db, departmentsByCode); err != nil {
		return fmt.Errorf("seed sample doctors: %w", err)
	}

	if err := seedLabTests(ctx, db); err != nil {
		return fmt.Errorf("seed lab tests: %w", err)
	}

	if err := seedMedicines(ctx, db); err != nil {
		return fmt.Errorf("seed medicines: %w", err)
	}

	if err := seedWardsAndBeds(ctx, db); err != nil {
		return fmt.Errorf("seed wards and beds: %w", err)
	}

	fmt.Println("seed complete")
	return nil
}

// sampleLabTests mirrors the blueprint's sample seed data (Section 11).
var sampleLabTests = []labtestsDomain.LabTest{
	{Code: "CBC", Name: "Complete Blood Count", Category: "Hematology", SampleType: "Blood", Unit: "cells/mcL", NormalRange: "4500-11000", Cost: 250},
	{Code: "HBA1C", Name: "HbA1c", Category: "Biochemistry", SampleType: "Blood", Unit: "%", NormalRange: "4.0-5.6", Cost: 450},
	{Code: "LFT", Name: "Liver Function Test", Category: "Biochemistry", SampleType: "Blood", Unit: "U/L", NormalRange: "varies by parameter", Cost: 600},
	{Code: "KFT", Name: "Kidney Function Test", Category: "Biochemistry", SampleType: "Blood", Unit: "mg/dL", NormalRange: "varies by parameter", Cost: 600},
	{Code: "LIPID", Name: "Lipid Profile", Category: "Biochemistry", SampleType: "Blood", Unit: "mg/dL", NormalRange: "varies by parameter", Cost: 500},
	{Code: "XRAYCHEST", Name: "X-Ray Chest", Category: "Radiology", SampleType: "N/A", Unit: "", NormalRange: "", Cost: 350},
}

// seedLabTests creates the sample lab test catalog entries if they don't
// already exist.
func seedLabTests(ctx context.Context, db *mongo.Database) error {
	testRepo := labtestsRepo.NewLabTestRepository(db)

	for _, t := range sampleLabTests {
		if _, err := testRepo.FindByCode(ctx, t.Code); err == nil {
			fmt.Printf("lab test already exists: %s (skipped)\n", t.Code)
			continue
		} else if !errors.Is(err, labtestsRepo.ErrNotFound) {
			return err
		}

		test := t
		test.Status = labtestsDomain.StatusActive
		if err := testRepo.Create(ctx, &test); err != nil {
			return fmt.Errorf("create lab test %s: %w", t.Code, err)
		}
		fmt.Printf("lab test created: %s (%s)\n", t.Name, t.Code)
	}

	return nil
}

// sampleMedicines mirrors the blueprint's sample seed data (Section 11).
var sampleMedicines = []pharmacyDomain.Medicine{
	{Name: "Paracetamol 500mg", GenericName: "Paracetamol", Category: "Analgesic", Unit: "Tablet", ReorderLevel: 100},
	{Name: "Amoxicillin 500mg", GenericName: "Amoxicillin", Category: "Antibiotic", Unit: "Capsule", ReorderLevel: 100},
	{Name: "Pantoprazole 40mg", GenericName: "Pantoprazole", Category: "Antacid", Unit: "Tablet", ReorderLevel: 50},
	{Name: "ORS", GenericName: "Oral Rehydration Salts", Category: "Electrolyte", Unit: "Sachet", ReorderLevel: 50},
}

// seedMedicines creates the sample medicine catalog entries if they don't
// already exist. It only creates the catalog entries (CurrentStock starts
// at 0, per CreateMedicine's own invariant) — receiving actual batches of
// stock is left to the pharmacy UI/API, the same way lab test catalog
// entries are seeded without any pre-seeded lab orders.
func seedMedicines(ctx context.Context, db *mongo.Database) error {
	medicineRepo := pharmacyRepo.NewMedicineRepository(db)

	for _, m := range sampleMedicines {
		if _, err := medicineRepo.FindByName(ctx, m.Name); err == nil {
			fmt.Printf("medicine already exists: %s (skipped)\n", m.Name)
			continue
		} else if !errors.Is(err, pharmacyRepo.ErrNotFound) {
			return err
		}

		medicine := m
		medicine.Status = pharmacyDomain.StatusActive
		if err := medicineRepo.Create(ctx, &medicine); err != nil {
			return fmt.Errorf("create medicine %s: %w", m.Name, err)
		}
		fmt.Printf("medicine created: %s\n", m.Name)
	}

	return nil
}

// sampleWard mirrors the blueprint's sample seed data (Section 11: "Wards:
// General Ward, Semi-Private, Private, ICU") plus a handful of beds per
// ward so the bed board has something to show immediately after seeding.
type sampleWard struct {
	name      string
	code      string
	bedCount  int
	bedPrefix string
}

var sampleWards = []sampleWard{
	{"General Ward", "GEN", 6, "GEN"},
	{"Semi-Private", "SEMI", 4, "SP"},
	{"Private", "PVT", 3, "PV"},
	{"ICU", "ICU", 3, "ICU"},
}

// seedWardsAndBeds creates the sample wards and their beds if they don't
// already exist.
func seedWardsAndBeds(ctx context.Context, db *mongo.Database) error {
	wardRepo := ipdRepo.NewWardRepository(db)
	bedRepo := ipdRepo.NewBedRepository(db)

	existingWards, err := wardRepo.List(ctx)
	if err != nil {
		return err
	}
	wardsByCode := make(map[string]ipdDomain.Ward, len(existingWards))
	for _, w := range existingWards {
		wardsByCode[w.Code] = w
	}

	for _, sw := range sampleWards {
		ward, ok := wardsByCode[sw.code]
		if !ok {
			ward = ipdDomain.Ward{Name: sw.name, Code: sw.code, Status: ipdDomain.StatusActive}
			if err := wardRepo.Create(ctx, &ward); err != nil {
				return fmt.Errorf("create ward %s: %w", sw.code, err)
			}
			fmt.Printf("ward created: %s (%s)\n", ward.Name, ward.Code)
		} else {
			fmt.Printf("ward already exists: %s (skipped)\n", sw.code)
		}

		existingBeds, err := bedRepo.List(ctx, ipdRepo.BedFilter{WardID: ward.ID.Hex()})
		if err != nil {
			return fmt.Errorf("list beds for ward %s: %w", sw.code, err)
		}
		bedNumbers := make(map[string]bool, len(existingBeds))
		for _, b := range existingBeds {
			bedNumbers[b.BedNumber] = true
		}

		for i := 1; i <= sw.bedCount; i++ {
			bedNumber := fmt.Sprintf("%s-%02d", sw.bedPrefix, i)
			if bedNumbers[bedNumber] {
				continue
			}
			bed := ipdDomain.Bed{WardID: ward.ID, WardName: ward.Name, BedNumber: bedNumber, Status: ipdDomain.BedAvailable}
			if err := bedRepo.Create(ctx, &bed); err != nil {
				return fmt.Errorf("create bed %s: %w", bedNumber, err)
			}
			fmt.Printf("bed created: %s (%s)\n", bedNumber, ward.Name)
		}
	}

	return nil
}

func seedRoles(ctx context.Context, db *mongo.Database) error {
	roleRepo := repository.NewRoleRepository(db)

	for _, code := range domain.AllRoleCodes {
		role := &domain.Role{
			Code:        code,
			Name:        domain.DisplayName(code),
			Permissions: domain.DefaultRolePermissions[code],
		}
		if err := roleRepo.Upsert(ctx, role); err != nil {
			return fmt.Errorf("upsert role %s: %w", code, err)
		}
		fmt.Printf("role ready: %-16s %d permissions\n", code, len(role.Permissions))
	}

	return nil
}

func seedAdminUser(ctx context.Context, db *mongo.Database) error {
	email := getEnv("SEED_ADMIN_EMAIL", "admin@hms.local")
	name := getEnv("SEED_ADMIN_NAME", "System Administrator")
	password := getEnv("SEED_ADMIN_PASSWORD", "ChangeMe123!")

	userRepo := repository.NewUserRepository(db)

	if _, err := userRepo.FindByEmail(ctx, email); err == nil {
		fmt.Printf("admin user already exists: %s (skipped)\n", email)
		return nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return err
	}

	admin := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		RoleCode:     domain.RoleAdmin,
		Status:       domain.UserStatusActive,
	}

	if err := userRepo.Create(ctx, admin); err != nil {
		return err
	}

	fmt.Printf("admin user created: %s\n", email)
	if password == "ChangeMe123!" {
		fmt.Println("WARNING: using the default seed password — set SEED_ADMIN_PASSWORD and re-seed, or change it via the password reset flow before going to production.")
	}

	return nil
}

// sampleDepartments mirrors the blueprint's sample seed data (Section 11).
var sampleDepartments = []departmentsDomain.Department{
	{Name: "Cardiology", Code: "CARD"},
	{Name: "General Medicine", Code: "GENMED"},
	{Name: "Pediatrics", Code: "PEDI"},
	{Name: "Laboratory", Code: "LAB"},
	{Name: "Pharmacy", Code: "PHARM"},
	{Name: "Emergency", Code: "ER"},
}

// seedDepartments creates the sample departments if they don't already
// exist and returns a code -> ID map for seedSampleDoctors to use.
func seedDepartments(ctx context.Context, db *mongo.Database) (map[string]primitive.ObjectID, error) {
	deptRepo := departmentsRepo.NewDepartmentRepository(db)
	byCode := make(map[string]primitive.ObjectID, len(sampleDepartments))

	for _, d := range sampleDepartments {
		existing, err := deptRepo.FindByCode(ctx, d.Code)
		if err == nil {
			byCode[d.Code] = existing.ID
			fmt.Printf("department already exists: %s (skipped)\n", d.Code)
			continue
		}
		if !errors.Is(err, departmentsRepo.ErrNotFound) {
			return nil, err
		}

		department := &departmentsDomain.Department{Name: d.Name, Code: d.Code, Status: departmentsDomain.StatusActive}
		if err := deptRepo.Create(ctx, department); err != nil {
			return nil, fmt.Errorf("create department %s: %w", d.Code, err)
		}
		byCode[d.Code] = department.ID
		fmt.Printf("department created: %s (%s)\n", d.Name, d.Code)
	}

	return byCode, nil
}

// sampleDoctor is one blueprint sample doctor (Section 11): a login
// account plus the clinical profile linked to it.
type sampleDoctor struct {
	name           string
	email          string
	departmentCode string
	licenseNo      string
	specialization string
	fee            float64
}

var sampleDoctors = []sampleDoctor{
	{"Dr. Anil Sharma", "anil.sharma@hms.local", "CARD", "LIC-CARD-001", "Cardiology", 800},
	{"Dr. Neha Verma", "neha.verma@hms.local", "GENMED", "LIC-GENMED-001", "General Medicine", 500},
	{"Dr. Rahul Mehta", "rahul.mehta@hms.local", "PEDI", "LIC-PEDI-001", "Pediatrics", 500},
}

// weekdayAvailability is a default Mon-Fri 09:00-17:00 schedule assigned to
// every seeded sample doctor.
func weekdayAvailability() []doctorsDomain.AvailabilitySlot {
	slots := make([]doctorsDomain.AvailabilitySlot, 0, 5)
	for day := 1; day <= 5; day++ { // Monday(1) .. Friday(5)
		slots = append(slots, doctorsDomain.AvailabilitySlot{DayOfWeek: day, StartTime: "09:00", EndTime: "17:00"})
	}
	return slots
}

// seedSampleDoctors creates the sample DOCTOR-role login accounts and
// their clinical profiles if they don't already exist. The default
// password is intentionally the same as the admin's dev default — change
// it (or delete these accounts) before any shared/production use.
func seedSampleDoctors(ctx context.Context, db *mongo.Database, departmentsByCode map[string]primitive.ObjectID) error {
	userRepo := repository.NewUserRepository(db)
	doctorRepo := doctorsRepo.NewDoctorRepository(db)
	password := getEnv("SEED_DOCTOR_PASSWORD", "ChangeMe123!")

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return err
	}

	for _, sd := range sampleDoctors {
		deptID, ok := departmentsByCode[sd.departmentCode]
		if !ok {
			return fmt.Errorf("department %s was not seeded", sd.departmentCode)
		}

		user, err := userRepo.FindByEmail(ctx, sd.email)
		if err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				return err
			}
			user = &domain.User{
				Name: sd.name, Email: sd.email, PasswordHash: passwordHash,
				RoleCode: domain.RoleDoctor, Status: domain.UserStatusActive,
			}
			if err := userRepo.Create(ctx, user); err != nil {
				return fmt.Errorf("create user for %s: %w", sd.email, err)
			}
			fmt.Printf("doctor user created: %s\n", sd.email)
		} else {
			fmt.Printf("doctor user already exists: %s (skipped)\n", sd.email)
		}

		if _, err := doctorRepo.FindByUserID(ctx, user.ID); err == nil {
			fmt.Printf("doctor profile already exists: %s (skipped)\n", sd.email)
			continue
		} else if !errors.Is(err, doctorsRepo.ErrNotFound) {
			return err
		}

		doctorProfile := &doctorsDomain.Doctor{
			UserID: user.ID, Name: sd.name, Email: sd.email,
			DepartmentID: deptID, DepartmentName: departmentNameForCode(sd.departmentCode),
			LicenseNo: sd.licenseNo, Specialization: sd.specialization, Fee: sd.fee,
			Availability: weekdayAvailability(), Status: doctorsDomain.StatusActive,
		}
		if err := doctorRepo.Create(ctx, doctorProfile); err != nil {
			return fmt.Errorf("create doctor profile for %s: %w", sd.email, err)
		}
		fmt.Printf("doctor profile created: %s (%s)\n", sd.name, sd.specialization)
	}

	return nil
}

func departmentNameForCode(code string) string {
	for _, d := range sampleDepartments {
		if d.Code == code {
			return d.Name
		}
	}
	return code
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
