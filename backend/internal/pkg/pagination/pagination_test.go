package pagination

import "testing"

func TestNewClampsInvalidInputs(t *testing.T) {
	cases := []struct {
		name             string
		page, pageSize   int
		wantPage, wantPS int
	}{
		{"zero values default", 0, 0, DefaultPage, DefaultPageSize},
		{"negative page defaults", -5, 10, DefaultPage, 10},
		{"negative page size defaults", 2, -1, 2, DefaultPageSize},
		{"page size above max is clamped", 1, 1000, 1, MaxPageSize},
		{"valid values pass through", 3, 50, 3, 50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := New(tc.page, tc.pageSize)
			if got.Page != tc.wantPage || got.PageSize != tc.wantPS {
				t.Errorf("New(%d, %d) = {%d, %d}, want {%d, %d}", tc.page, tc.pageSize, got.Page, got.PageSize, tc.wantPage, tc.wantPS)
			}
		})
	}
}

func TestSkipAndLimit(t *testing.T) {
	req := New(3, 20)
	if req.Skip() != 40 {
		t.Errorf("expected Skip() of 40 for page 3 / pageSize 20, got %d", req.Skip())
	}
	if req.Limit() != 20 {
		t.Errorf("expected Limit() of 20, got %d", req.Limit())
	}
}

func TestNewResultComputesTotalPagesAndNeverReturnsNilItems(t *testing.T) {
	req := New(1, 10)

	result := NewResult[string](nil, req, 25)
	if result.Items == nil {
		t.Errorf("expected NewResult to normalize nil items to an empty slice")
	}
	if result.TotalPages != 3 {
		t.Errorf("expected 3 total pages for 25 items at page size 10, got %d", result.TotalPages)
	}

	exact := NewResult([]string{"a", "b"}, New(1, 10), 20)
	if exact.TotalPages != 2 {
		t.Errorf("expected 2 total pages for an exact multiple (20 items / 10 per page), got %d", exact.TotalPages)
	}
}
