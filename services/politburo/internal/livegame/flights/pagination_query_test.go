package flights

import (
	"net/url"
	"testing"
)

func TestParsePaginationQueryDefaults(t *testing.T) {
	pageNumber, pageLength, err := ParsePaginationQuery(url.Values{})
	if err != nil || pageNumber != DefaultPageNumber || pageLength != DefaultPageLength {
		t.Fatalf("pageNumber=%d pageLength=%d err=%v", pageNumber, pageLength, err)
	}
}

func TestParsePaginationQueryRejectsOversizePageLength(t *testing.T) {
	_, _, err := ParsePaginationQuery(url.Values{"pageLength": []string{"5001"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePaginationQueryAllowsMaxPageLength(t *testing.T) {
	_, pageLength, err := ParsePaginationQuery(url.Values{"pageLength": []string{"5000"}})
	if err != nil || pageLength != MaxPageLength {
		t.Fatalf("pageLength=%d err=%v", pageLength, err)
	}
}
