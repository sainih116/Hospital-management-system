package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Role codes for the six primary operational roles defined by the HMS
// blueprint's RBAC matrix.
const (
	RoleAdmin        = "ADMIN"
	RoleDoctor       = "DOCTOR"
	RoleNurse        = "NURSE"
	RoleReceptionist = "RECEPTIONIST"
	RolePharmacist   = "PHARMACIST"
	RoleLabTech      = "LAB_TECHNICIAN"
)

// AllRoleCodes lists every valid role code, used for request validation and
// seeding.
var AllRoleCodes = []string{RoleAdmin, RoleDoctor, RoleNurse, RoleReceptionist, RolePharmacist, RoleLabTech}

// Role is a named, permissioned group that users are assigned to.
type Role struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Code        string             `bson:"code"`
	Name        string             `bson:"name"`
	Permissions []string           `bson:"permissions"`
	CreatedAt   time.Time          `bson:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"`
}

// HasPermission reports whether the role grants the given permission code.
func (r *Role) HasPermission(code string) bool {
	for _, p := range r.Permissions {
		if p == code {
			return true
		}
	}
	return false
}

// roleDisplayNames maps role codes to their human-readable names, used by
// the seed command.
var roleDisplayNames = map[string]string{
	RoleAdmin:        "Administrator",
	RoleDoctor:       "Doctor",
	RoleNurse:        "Nurse",
	RoleReceptionist: "Receptionist",
	RolePharmacist:   "Pharmacist",
	RoleLabTech:      "Lab Technician",
}

// DisplayName returns the human-readable name for a role code.
func DisplayName(code string) string {
	if name, ok := roleDisplayNames[code]; ok {
		return name
	}
	return code
}

// IsValidRoleCode reports whether code is one of the known role codes.
func IsValidRoleCode(code string) bool {
	for _, c := range AllRoleCodes {
		if c == code {
			return true
		}
	}
	return false
}
