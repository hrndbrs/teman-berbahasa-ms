package pagination

import "strconv"

const (
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// ParsePage parses page/per_page query params. Returns page >= 1 and perPage
// clamped to [1, MaxPerPage]. Invalid or missing values fall back to defaults.
func ParsePage(pageStr, perPageStr string) (page, perPage int) {
	page = 1
	perPage = DefaultPerPage
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
		perPage = pp
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return
}

// TotalPages returns ⌈total / perPage⌉. Returns 0 for empty result sets.
func TotalPages(total int64, perPage int) int {
	if total == 0 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}
