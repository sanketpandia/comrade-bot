package flights

import (
	"fmt"
	"net/url"
	"strconv"
)

// ParsePaginationQuery reads pageNumber and pageLength from query values, applying defaults and limits.
func ParsePaginationQuery(values url.Values) (pageNumber, pageLength int, err error) {
	pageNumber = DefaultPageNumber
	pageLength = DefaultPageLength

	if raw := values.Get("pageNumber"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("pageNumber must be an integer")
		}
		pageNumber = parsed
	}
	if raw := values.Get("pageLength"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("pageLength must be an integer")
		}
		pageLength = parsed
	}

	if pageNumber < 1 {
		return 0, 0, fmt.Errorf("pageNumber must be at least 1")
	}
	if pageLength < 1 {
		return 0, 0, fmt.Errorf("pageLength must be at least 1")
	}
	if pageLength > MaxPageLength {
		return 0, 0, fmt.Errorf("pageLength must be at most %d", MaxPageLength)
	}
	return pageNumber, pageLength, nil
}
