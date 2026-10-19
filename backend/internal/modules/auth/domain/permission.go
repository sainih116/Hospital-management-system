package domain

// Permission codes. Every module in the system protects its endpoints with
// one or more of these codes via middleware.RequirePermission. New modules
// add their own codes here as they are built, keeping the full permission
// registry in one place.
const (
	PermDashboardView = "dashboard.view"

	PermUserManage = "user.manage"

	PermDepartmentManage = "department.manage"

	PermDoctorRead   = "doctor.read"
	PermDoctorManage = "doctor.manage"

	PermPatientCreate = "patient.create"
	PermPatientRead   = "patient.read"
	PermPatientUpdate = "patient.update"
	PermPatientDelete = "patient.delete"

	PermMedicalRecordRead  = "emr.read"
	PermMedicalRecordWrite = "emr.write"

	PermPrescriptionRead  = "prescription.read"
	PermPrescriptionWrite = "prescription.write"

	PermAppointmentView   = "appointment.view"
	PermAppointmentManage = "appointment.manage"

	PermLabTestManage = "labtest.manage"

	PermLabOrderRead    = "lab.order.read"
	PermLabOrderWrite   = "lab.order.write"
	PermLabResultVerify = "lab.result.verify"

	PermPharmacyRead     = "pharmacy.read"
	PermPharmacyDispense = "pharmacy.dispense"

	PermBillingRead   = "billing.read"
	PermBillingManage = "billing.manage"

	PermIPDRead  = "ipd.read"
	PermIPDWrite = "ipd.write"

	PermHRManage       = "hr.manage"
	PermSettingsManage = "settings.manage"

	PermReportExport = "report.export"
)

// DefaultRolePermissions is the seed source of truth for the RBAC matrix:
// Admin has full access; every other role gets exactly the permissions the
// blueprint's role matrix grants it.
var DefaultRolePermissions = map[string][]string{
	RoleAdmin: {
		PermDashboardView,
		PermUserManage,
		PermDepartmentManage,
		PermDoctorRead, PermDoctorManage,
		PermPatientCreate, PermPatientRead, PermPatientUpdate, PermPatientDelete,
		PermMedicalRecordRead, PermMedicalRecordWrite,
		PermPrescriptionRead, PermPrescriptionWrite,
		PermAppointmentView, PermAppointmentManage,
		PermLabTestManage,
		PermLabOrderRead, PermLabOrderWrite, PermLabResultVerify,
		PermPharmacyRead, PermPharmacyDispense,
		PermBillingRead, PermBillingManage,
		PermIPDRead, PermIPDWrite,
		PermHRManage,
		PermSettingsManage,
		PermReportExport,
	},
	RoleDoctor: {
		PermDashboardView,
		PermDoctorRead,
		PermPatientRead,
		PermMedicalRecordRead, PermMedicalRecordWrite,
		PermPrescriptionRead, PermPrescriptionWrite,
		PermAppointmentView,
		PermLabOrderRead, PermLabOrderWrite,
		PermIPDRead, PermIPDWrite,
	},
	RoleNurse: {
		PermDashboardView,
		PermDoctorRead,
		PermPatientRead,
		PermMedicalRecordRead,
		PermPrescriptionRead,
		PermIPDRead, PermIPDWrite,
	},
	RoleReceptionist: {
		PermDashboardView,
		PermDoctorRead,
		PermPatientCreate, PermPatientRead,
		PermPrescriptionRead,
		PermAppointmentView, PermAppointmentManage,
		PermBillingRead,
		PermIPDRead,
	},
	RolePharmacist: {
		PermDashboardView,
		PermPatientRead,
		PermPrescriptionRead,
		PermPharmacyRead, PermPharmacyDispense,
	},
	RoleLabTech: {
		PermDashboardView,
		PermPatientRead,
		PermMedicalRecordRead,
		PermLabOrderRead, PermLabOrderWrite, PermLabResultVerify,
	},
}
