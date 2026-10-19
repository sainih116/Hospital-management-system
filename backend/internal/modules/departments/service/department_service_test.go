package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/modules/departments/dto"
)

func newTestService() (DepartmentService, *fakeDepartmentRepository, *fakeAuditRecorder) {
	repo := newFakeDepartmentRepository()
	auditRecorder := &fakeAuditRecorder{}
	svc := NewDepartmentService(Deps{Departments: repo, Audit: auditRecorder, Logger: zap.NewNop()})
	return svc, repo, auditRecorder
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func TestCreateDepartmentSucceeds(t *testing.T) {
	svc, _, auditRecorder := newTestService()

	dept, err := svc.Create(context.Background(), dto.CreateDepartmentRequest{Name: "Cardiology", Code: "CARD"}, audit.Actor{UserID: "admin1"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if dept.Status != "ACTIVE" {
		t.Errorf("expected new department to be ACTIVE, got %s", dept.Status)
	}
	if actions := auditRecorder.actions(); len(actions) != 1 || actions[0] != "DEPARTMENT_CREATED" {
		t.Fatalf("expected a DEPARTMENT_CREATED audit entry, got %v", actions)
	}
}

func TestCreateDepartmentRejectsDuplicateCode(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Create(ctx, dto.CreateDepartmentRequest{Name: "Cardiology", Code: "CARD"}, audit.Actor{}); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}

	_, err := svc.Create(ctx, dto.CreateDepartmentRequest{Name: "Cardiac Care", Code: "CARD"}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for a duplicate department code")
	}
	if code := appErrCode(t, err); code != "DEPARTMENT_CODE_TAKEN" {
		t.Fatalf("expected DEPARTMENT_CODE_TAKEN, got %s", code)
	}
}

func TestUpdateDepartmentCanDeactivate(t *testing.T) {
	svc, _, auditRecorder := newTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, dto.CreateDepartmentRequest{Name: "Emergency", Code: "ER"}, audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	inactive := "INACTIVE"
	updated, err := svc.Update(ctx, created.ID, dto.UpdateDepartmentRequest{Status: &inactive}, audit.Actor{})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != "INACTIVE" {
		t.Errorf("expected status INACTIVE, got %s", updated.Status)
	}
	if actions := auditRecorder.actions(); len(actions) != 2 || actions[1] != "DEPARTMENT_UPDATED" {
		t.Fatalf("expected a DEPARTMENT_UPDATED audit entry, got %v", actions)
	}
}

func TestUpdateDepartmentRejectsEmptyBody(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, dto.CreateDepartmentRequest{Name: "Pediatrics", Code: "PED"}, audit.Actor{})

	_, err := svc.Update(ctx, created.ID, dto.UpdateDepartmentRequest{}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an update with no fields")
	}
}

func TestUpdateDepartmentReturnsNotFoundForUnknownID(t *testing.T) {
	svc, _, _ := newTestService()
	name := "Something"

	_, err := svc.Update(context.Background(), "000000000000000000000000", dto.UpdateDepartmentRequest{Name: &name}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an unknown department id")
	}
	if code := appErrCode(t, err); code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %s", code)
	}
}

func TestListReturnsAllDepartments(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Create(ctx, dto.CreateDepartmentRequest{Name: "Cardiology", Code: "CARD"}, audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := svc.Create(ctx, dto.CreateDepartmentRequest{Name: "Pediatrics", Code: "PED"}, audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 departments, got %d", len(list))
	}
}
