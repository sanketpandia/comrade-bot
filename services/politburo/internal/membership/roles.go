package membership

const (
	RoleProletariat    = "proletariat"
	RoleBourgeoisie    = "bourgeoisie"
	RoleAdministrator  = "administrator"
)

func IsMemberRole(role string) bool {
	return role == RoleProletariat || role == RoleBourgeoisie || role == RoleAdministrator
}

func IsStaffRole(role string) bool {
	return role == RoleBourgeoisie || role == RoleAdministrator
}

func IsAdministratorRole(role string) bool {
	return role == RoleAdministrator
}
