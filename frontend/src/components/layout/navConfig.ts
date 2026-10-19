import DashboardRoundedIcon from "@mui/icons-material/DashboardRounded";
import EventAvailableRoundedIcon from "@mui/icons-material/EventAvailableRounded";
import GroupsRoundedIcon from "@mui/icons-material/GroupsRounded";
import HistoryRoundedIcon from "@mui/icons-material/HistoryRounded";
import LocalPharmacyRoundedIcon from "@mui/icons-material/LocalPharmacyRounded";
import ManageAccountsRoundedIcon from "@mui/icons-material/ManageAccountsRounded";
import MedicalServicesRoundedIcon from "@mui/icons-material/MedicalServicesRounded";
import MedicationRoundedIcon from "@mui/icons-material/MedicationRounded";
import PaidRoundedIcon from "@mui/icons-material/PaidRounded";
import PeopleAltRoundedIcon from "@mui/icons-material/PeopleAltRounded";
import ScienceRoundedIcon from "@mui/icons-material/ScienceRounded";
import SettingsRoundedIcon from "@mui/icons-material/SettingsRounded";
import BedRoundedIcon from "@mui/icons-material/BedRounded";
import AssessmentRoundedIcon from "@mui/icons-material/AssessmentRounded";

import type { SvgIconComponent } from "@/types/mui";
import type { RoleCode } from "@/types/roles";
import { ROLES } from "@/types/roles";

export interface NavItem {
  label: string;
  path: string;
  icon: SvgIconComponent;
  /** Roles allowed to see this item. Omit to allow every authenticated role. */
  roles?: RoleCode[];
}

export const NAV_ITEMS: NavItem[] = [
  { label: "Dashboard", path: "/dashboard", icon: DashboardRoundedIcon },
  {
    label: "Patients",
    path: "/patients",
    icon: PeopleAltRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.DOCTOR, ROLES.NURSE, ROLES.RECEPTIONIST, ROLES.PHARMACIST, ROLES.LAB_TECHNICIAN],
  },
  {
    label: "Doctors",
    path: "/doctors",
    icon: GroupsRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.DOCTOR, ROLES.NURSE, ROLES.RECEPTIONIST],
  },
  {
    label: "Appointments",
    path: "/appointments",
    icon: EventAvailableRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.DOCTOR, ROLES.RECEPTIONIST],
  },
  {
    label: "Medical Records",
    path: "/emr",
    icon: MedicalServicesRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.DOCTOR, ROLES.NURSE, ROLES.LAB_TECHNICIAN],
  },
  {
    label: "Prescriptions",
    path: "/prescriptions",
    icon: MedicationRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.DOCTOR, ROLES.NURSE, ROLES.RECEPTIONIST, ROLES.PHARMACIST],
  },
  {
    label: "Laboratory",
    path: "/laboratory",
    icon: ScienceRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.DOCTOR, ROLES.LAB_TECHNICIAN],
  },
  {
    label: "Pharmacy",
    path: "/pharmacy",
    icon: LocalPharmacyRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.PHARMACIST],
  },
  {
    label: "IPD / Wards",
    path: "/ipd",
    icon: BedRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.DOCTOR, ROLES.NURSE, ROLES.RECEPTIONIST],
  },
  {
    label: "Billing",
    path: "/billing",
    icon: PaidRoundedIcon,
    roles: [ROLES.ADMIN, ROLES.RECEPTIONIST],
  },
  {
    label: "Reports",
    path: "/reports",
    icon: AssessmentRoundedIcon,
    roles: [ROLES.ADMIN],
  },
  {
    // Settings sub-pages (Departments, Lab Tests, ...) each get their own
    // nav item rather than a landing hub page, until there are enough of
    // them to justify one.
    label: "Departments",
    path: "/settings/departments",
    icon: SettingsRoundedIcon,
    roles: [ROLES.ADMIN],
  },
  {
    label: "Lab Test Catalog",
    path: "/settings/lab-tests",
    icon: ScienceRoundedIcon,
    roles: [ROLES.ADMIN],
  },
  {
    label: "Audit Logs",
    path: "/settings/audit-logs",
    icon: HistoryRoundedIcon,
    roles: [ROLES.ADMIN],
  },
  {
    label: "Staff / Users",
    path: "/settings/users",
    icon: ManageAccountsRoundedIcon,
    roles: [ROLES.ADMIN],
  },
];
