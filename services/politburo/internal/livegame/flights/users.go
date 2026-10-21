package flights

// AppendUserIDIfNew returns users with userID appended when userID is non-empty and not already present.
func AppendUserIDIfNew(users []string, userID string) []string {
	if userID == "" {
		return users
	}
	for _, id := range users {
		if id == userID {
			return users
		}
	}
	return append(users, userID)
}

// UserIDsEqual reports whether two user ID slices are identical.
func UserIDsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
