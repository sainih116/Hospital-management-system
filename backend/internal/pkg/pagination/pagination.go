// Package pagination provides a single, consistent page/pageSize request
// and result shape shared by every module's list/search endpoints.
package pagination

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Request is a normalized page request. Use New to build one from raw,
// possibly zero-valued query parameters.
type Request struct {
	Page     int
	PageSize int
}

// New builds a Request, clamping page to at least 1 and pageSize to
// [1, MaxPageSize], defaulting either when zero.
func New(page, pageSize int) Request {
	if page < 1 {
		page = DefaultPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return Request{Page: page, PageSize: pageSize}
}

// Skip returns the number of documents to skip for this page.
func (r Request) Skip() int64 {
	return int64((r.Page - 1) * r.PageSize)
}

// Limit returns the page size as an int64, as MongoDB's driver expects.
func (r Request) Limit() int64 {
	return int64(r.PageSize)
}

// Result is the standard paginated list envelope returned by search/list
// endpoints across every module.
type Result[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// NewResult builds a Result from a page of items, the request that
// produced them, and the total matching document count.
func NewResult[T any](items []T, req Request, total int64) Result[T] {
	totalPages := 0
	if req.PageSize > 0 {
		totalPages = int((total + int64(req.PageSize) - 1) / int64(req.PageSize))
	}

	if items == nil {
		items = []T{}
	}

	return Result[T]{
		Items:      items,
		Page:       req.Page,
		PageSize:   req.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}
