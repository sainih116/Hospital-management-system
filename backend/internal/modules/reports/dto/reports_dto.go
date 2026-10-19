// Package dto defines the reporting module's request/response payloads.
package dto

// DateRangeQuery is bound from query parameters for every trend/breakdown
// endpoint. From/To are "YYYY-MM-DD" dates (parsed by the service); both
// default to the last 30 days when omitted.
type DateRangeQuery struct {
	From string `form:"from"`
	To   string `form:"to"`
}

// DoctorWorkloadQuery adds a result-size cap to DateRangeQuery.
type DoctorWorkloadQuery struct {
	From  string `form:"from"`
	To    string `form:"to"`
	Limit int    `form:"limit"`
}

// SummaryResponse is the dashboard's top-level snapshot of cards.
type SummaryResponse struct {
	TotalPatients           int64   `json:"totalPatients"`
	TotalDoctors            int64   `json:"totalDoctors"`
	AppointmentsToday       int64   `json:"appointmentsToday"`
	TotalRevenueCollected   float64 `json:"totalRevenueCollected"`
	OutstandingBalance      float64 `json:"outstandingBalance"`
	LowStockMedicineCount   int64   `json:"lowStockMedicineCount"`
	PendingLabVerifications int64   `json:"pendingLabVerifications"`
	ActivePrescriptions     int64   `json:"activePrescriptions"`
}

// StatusCountResponse is one (status, count) bucket.
type StatusCountResponse struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// TrendPointResponse is one (date, value) point on a time-series chart.
type TrendPointResponse struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// DoctorWorkloadResponse is one doctor's appointment count within a date
// range.
type DoctorWorkloadResponse struct {
	DoctorID   string `json:"doctorId"`
	DoctorName string `json:"doctorName"`
	Count      int64  `json:"count"`
}

// NameCountResponse is one (name, count) bucket — used for lab test
// volume.
type NameCountResponse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
