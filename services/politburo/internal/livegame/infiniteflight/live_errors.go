package infiniteflight

import "errors"

// ErrLiveOrganizationNotFound means the upstream API has no such org for this token.
var ErrLiveOrganizationNotFound = errors.New("live organization not found")
