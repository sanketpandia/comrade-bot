package flights

import "strings"

func Username(obs Observation) string {
	return UsernameFromObservation(obs)
}

func ContainsFold(value, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}
