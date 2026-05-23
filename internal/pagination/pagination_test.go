package pagination_test

import (
	"testing"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/pagination"
)

func TestParsePage_Defaults(t *testing.T) {
	page, perPage := pagination.ParsePage("", "")
	if page != 1 {
		t.Errorf("expected page=1, got %d", page)
	}
	if perPage != pagination.DefaultPerPage {
		t.Errorf("expected perPage=%d, got %d", pagination.DefaultPerPage, perPage)
	}
}

func TestParsePage_InvalidInput(t *testing.T) {
	page, perPage := pagination.ParsePage("abc", "xyz")
	if page != 1 {
		t.Errorf("expected page=1 on invalid input, got %d", page)
	}
	if perPage != pagination.DefaultPerPage {
		t.Errorf("expected perPage=%d on invalid input, got %d", pagination.DefaultPerPage, perPage)
	}
}

func TestParsePage_ZeroAndNegative(t *testing.T) {
	page, perPage := pagination.ParsePage("0", "-5")
	if page != 1 {
		t.Errorf("expected page=1 for page=0, got %d", page)
	}
	if perPage != pagination.DefaultPerPage {
		t.Errorf("expected perPage=%d for perPage=-5, got %d", pagination.DefaultPerPage, perPage)
	}
}

func TestParsePage_CapsPerPage(t *testing.T) {
	_, perPage := pagination.ParsePage("1", "999")
	if perPage != pagination.MaxPerPage {
		t.Errorf("expected perPage capped at %d, got %d", pagination.MaxPerPage, perPage)
	}
}

func TestParsePage_ValidValues(t *testing.T) {
	page, perPage := pagination.ParsePage("3", "50")
	if page != 3 {
		t.Errorf("expected page=3, got %d", page)
	}
	if perPage != 50 {
		t.Errorf("expected perPage=50, got %d", perPage)
	}
}

func TestTotalPages_Empty(t *testing.T) {
	if n := pagination.TotalPages(0, 20); n != 0 {
		t.Errorf("expected 0 pages for empty result, got %d", n)
	}
}

func TestTotalPages_Exact(t *testing.T) {
	if n := pagination.TotalPages(40, 20); n != 2 {
		t.Errorf("expected 2 pages for 40/20, got %d", n)
	}
}

func TestTotalPages_Ceil(t *testing.T) {
	if n := pagination.TotalPages(41, 20); n != 3 {
		t.Errorf("expected 3 pages for 41/20 (ceiling), got %d", n)
	}
}

func TestTotalPages_SingleItem(t *testing.T) {
	if n := pagination.TotalPages(1, 20); n != 1 {
		t.Errorf("expected 1 page for 1 item, got %d", n)
	}
}
