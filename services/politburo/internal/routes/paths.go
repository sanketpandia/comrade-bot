package routes

import "strings"

// Prefix returns the path prefix for chi/OpenAPI patterns (everything before the first '{').
func Prefix(pattern string) string {
	if i := strings.IndexByte(pattern, '{'); i >= 0 {
		return pattern[:i]
	}
	return pattern
}

// Group prefixes for HasPrefix checks (not single routes).
const (
	APIPrefix          = "/api/"
	APIV1              = "/api/v1"
	APIV1Prefix        = "/api/v1/"
	APIV1GamePrefix    = "/api/v1/game/"
	APIV1VAPrefix      = "/api/v1/va/"
	APIV1OperatorPrefix = "/api/v1/operator/"
	HealthPrefix       = "/health/"
	MetricsPrefix      = "/metrics/"
)

const (
	Home = "/"

	AuthLogin  = "/auth/login"
	AuthLogout = "/auth/logout"

	Dashboard   = "/dashboard"
	Admin       = "/admin"
	AdminGeneral = "/admin/general"

	OperatorReports              = "/operator/reports"
	OperatorReportsResolve       = "/operator/reports/{reportID}/resolve"
	OperatorReportsResolvePrefix = "/operator/reports/"

	MapsFlightsActive           = "/maps/flights/active"
	MapsFlightsActiveServers    = "/maps/flights/active/servers"
	MapsFlightsActiveFlights    = "/maps/flights/active/flights"
	MapsFlightsActiveFlightDetail = "/maps/flights/active/flights/{flightID}"

	TermsConditions = "/terms-conditions"
	Privacy         = "/privacy"
	Contact         = "/contact"

	StaticPrefix      = "/static/"
	StaticBrandPrefix = "/static/brand/"

	Metrics = "/metrics"
	Health  = "/health"
	HealthStatus = "/health/status"

	OAuthCallback = "/oauth/callback"

	GameSessionsActive              = "/api/v1/game/sessions/active"
	GameFlightsActive               = "/api/v1/game/flights/active/{normalizedServerName}"
	GameFlightActive                = "/api/v1/game/flight/active/{flightId}"
	SignedLink                      = "/api/v1/signed-link"
	OAuthInfiniteFlight             = "/api/v1/oauth/infinite-flight"
	UserMe                          = "/api/v1/user/me"
	UsersCreate                     = "/api/v1/users"
	ServersCreate                   = "/api/v1/servers"
	AdminVerifyGod                  = "/api/v1/admin/verify-god"
	AdminBanUser                    = "/api/v1/admin/users/ban"
	VA                              = "/api/v1/va"
	VAConfig                        = "/api/v1/va/config"
	VAWebhookTrigger                = "/api/v1/va/webhook/trigger"

	UserStatus           = "/api/v1/user/status"
	MembershipsJoin      = "/api/v1/memberships/join"
	ReportsOccupiedIFC     = "/api/v1/reports/occupied-ifc"
	APIOperatorReports   = "/api/v1/operator/reports"
	APIOperatorReportsResolve = "/api/v1/operator/reports/{reportID}/resolve"
)

// OpenAPIPaths lists every path key in openapi/politburo/openapi.yaml (must stay in sync).
var OpenAPIPaths = []string{
	HealthStatus,
	GameSessionsActive,
	GameFlightsActive,
	GameFlightActive,
	SignedLink,
	OAuthInfiniteFlight,
	OAuthCallback,
	UserMe,
	UsersCreate,
	ServersCreate,
	AdminVerifyGod,
	AdminBanUser,
	VA,
	VAConfig,
	VAWebhookTrigger,
}
