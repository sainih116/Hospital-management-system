package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"hms-backend/internal/apperror"
	"hms-backend/internal/audit"
	"hms-backend/internal/modules/labtests/dto"
)

func newTestService() (LabTestService, *fakeAuditRecorder) {
	auditRecorder := &fakeAuditRecorder{}
	svc := NewLabTestService(Deps{Tests: newFakeLabTestRepository(), Audit: auditRecorder, Logger: zap.NewNop()})
	return svc, auditRecorder
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected an *apperror.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

func TestCreateLabTestSucceeds(t *testing.T) {
	svc, auditRecorder := newTestService()

	test, err := svc.Create(context.Background(), dto.CreateLabTestRequest{
		Code: "CBC", Name: "Complete Blood Count", Category: "Hematology", SampleType: "Blood",
		Unit: "cells/mcL", NormalRange: "4500-11000", Cost: 250,
	}, audit.Actor{UserID: "admin1"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if test.Status != "ACTIVE" {
		t.Errorf("expected new test to be ACTIVE, got %s", test.Status)
	}
	if actions := auditRecorder.actions(); len(actions) != 1 || actions[0] != "LAB_TEST_CREATED" {
		t.Fatalf("expected a LAB_TEST_CREATED audit entry, got %v", actions)
	}
}

func TestCreateLabTestRejectsDuplicateCode(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	req := dto.CreateLabTestRequest{Code: "CBC", Name: "Complete Blood Count", Cost: 250}

	if _, err := svc.Create(ctx, req, audit.Actor{}); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}

	_, err := svc.Create(ctx, dto.CreateLabTestRequest{Code: "CBC", Name: "Duplicate", Cost: 100}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for a duplicate test code")
	}
	if code := appErrCode(t, err); code != "LAB_TEST_CODE_TAKEN" {
		t.Fatalf("expected LAB_TEST_CODE_TAKEN, got %s", code)
	}
}

func TestUpdateLabTestCanDeactivate(t *testing.T) {
	svc, auditRecorder := newTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, dto.CreateLabTestRequest{Code: "LFT", Name: "Liver Function Test", Cost: 400}, audit.Actor{})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	inactive := "INACTIVE"
	updated, err := svc.Update(ctx, created.ID, dto.UpdateLabTestRequest{Status: &inactive}, audit.Actor{})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != "INACTIVE" {
		t.Errorf("expected status INACTIVE, got %s", updated.Status)
	}
	if actions := auditRecorder.actions(); len(actions) != 2 || actions[1] != "LAB_TEST_UPDATED" {
		t.Fatalf("expected a LAB_TEST_UPDATED audit entry, got %v", actions)
	}
}

func TestUpdateLabTestRejectsEmptyBody(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	created, _ := svc.Create(ctx, dto.CreateLabTestRequest{Code: "KFT", Name: "Kidney Function Test", Cost: 400}, audit.Actor{})

	_, err := svc.Update(ctx, created.ID, dto.UpdateLabTestRequest{}, audit.Actor{})
	if err == nil {
		t.Fatalf("expected an error for an update with no fields")
	}
}

func TestListReturnsAllLabTests(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Create(ctx, dto.CreateLabTestRequest{Code: "CBC", Name: "Complete Blood Count", Cost: 250}, audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := svc.Create(ctx, dto.CreateLabTestRequest{Code: "HBA1C", Name: "HbA1c", Cost: 300}, audit.Actor{}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 lab tests, got %d", len(list))
	}
}
