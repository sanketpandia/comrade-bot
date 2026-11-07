package http

import (
	stdhttp "net/http"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/app"
	"infinite-experiment/politburo/internal/routes"
	"infinite-experiment/politburo/internal/transport/http/response"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func mountAPI(router chi.Router, h handlers, application *app.App) {
	stacks := apiStacks{
		bot:         botStack(application),
		god:         godStack(application),
		vaAdmin:     vaAdminStack(application),
		vaStaff:     vaStaffStack(application),
		vaMember:    vaMemberStack(application),
		currentUser: currentUserStack(application),
	}
	api := apiHandler{handlers: h, stacks: stacks}

	// Public ops + OpenAPI machine API (paths include /health/* and /api/v1/*).
	// Game GETs accept a session cookie or API key; other /api/v1 paths need a key.
	politburoapi.HandlerWithOptions(api, politburoapi.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: writeParameterError,
	})
	router.Handle(routes.Metrics, promhttp.HandlerFor(application.Metrics.Prometheus, promhttp.HandlerOpts{}))
}

type apiStacks struct {
	bot, god, vaAdmin, vaStaff, vaMember, currentUser middlewareStack
}

type apiHandler struct {
	handlers handlers
	stacks   apiStacks
}

func (h apiHandler) GetHealthStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.handlers.health.GetStatus(w, r)
}

func (h apiHandler) GetActiveSessions(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.handlers.sessions.GetActiveSessions(w, r)
}

func (h apiHandler) GetActiveFlights(w stdhttp.ResponseWriter, r *stdhttp.Request, normalizedServerName string, params politburoapi.GetActiveFlightsParams) {
	h.handlers.flights.GetActiveFlights(w, r, normalizedServerName, params)
}

func (h apiHandler) GetActiveFlight(w stdhttp.ResponseWriter, r *stdhttp.Request, flightID openapi_types.UUID) {
	h.handlers.flights.GetActiveFlight(w, r, flightID)
}

func (h apiHandler) GetActiveFlightPlan(w stdhttp.ResponseWriter, r *stdhttp.Request, flightID openapi_types.UUID) {
	h.handlers.flights.GetActiveFlightPlan(w, r, flightID)
}

func (h apiHandler) GetActiveFlightPositionHistory(w stdhttp.ResponseWriter, r *stdhttp.Request, flightID openapi_types.UUID) {
	h.handlers.flights.GetActiveFlightPositionHistory(w, r, flightID)
}

func (h apiHandler) GenerateSignedLink(w stdhttp.ResponseWriter, r *stdhttp.Request, _ politburoapi.GenerateSignedLinkParams) {
	h.handlers.signedLink.GenerateSignedLink(w, r)
}

func (h apiHandler) CreateUser(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.CreateUserParams) {
	h.stacks.bot.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.identity.CreateUser(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) CreateServer(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.CreateServerParams) {
	h.stacks.bot.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.identity.CreateServer(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) LinkMembership(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.LinkMembershipParams) {
	h.stacks.bot.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.identity.LinkMembership(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) VerifyGodMode(w stdhttp.ResponseWriter, r *stdhttp.Request, _ politburoapi.VerifyGodModeParams) {
	h.stacks.god.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.identity.VerifyGod(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) BanUser(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.BanUserParams) {
	h.stacks.god.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.identity.BanUser(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) GetVA(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.va.GetVA(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListVAMembers(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.ListVAMembersParams) {
	h.stacks.vaStaff.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.va.ListVAMembers(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) UpdateVAMemberRole(w stdhttp.ResponseWriter, r *stdhttp.Request, userID openapi_types.UUID) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.va.UpdateVAMemberRole(w, r, userID)
	})).ServeHTTP(w, r)
}

func (h apiHandler) RemoveVAMember(w stdhttp.ResponseWriter, r *stdhttp.Request, userID openapi_types.UUID) {
	h.stacks.vaStaff.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.va.RemoveVAMember(w, r, userID)
	})).ServeHTTP(w, r)
}

func (h apiHandler) GetVAConfig(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.GetVAConfigParams) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if !params.ConfigType.Valid() {
			response.WriteError(w, stdhttp.StatusBadRequest, "UNKNOWN_CONFIG_TYPE", "unknown config type")
			return
		}
		h.handlers.va.GetVAConfig(w, r, string(params.ConfigType))
	})).ServeHTTP(w, r)
}

func (h apiHandler) UpsertVAConfig(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.UpsertVAConfigParams) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.va.UpsertVAConfig(w, r, string(params.ConfigType))
	})).ServeHTTP(w, r)
}

func (h apiHandler) TriggerVAWebhook(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.va.TriggerVAWebhook(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) GetCurrentUser(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.stacks.currentUser.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.user.GetMe(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) GetInfiniteFlightOAuthAuthorize(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.GetInfiniteFlightOAuthAuthorizeParams) {
	h.stacks.currentUser.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		scopeSet := ""
		if params.ScopeSet != nil {
			scopeSet = string(*params.ScopeSet)
		}
		h.handlers.oauth.GetAuthorizeURL(w, r, scopeSet)
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListLiveOrganizations(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.live.ListOrganizations(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) CompleteInfiniteFlightOAuthCallback(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.CompleteInfiniteFlightOAuthCallbackParams) {
	h.handlers.oauth.CompleteCallback(w, r, params.Code, params.State)
}

func (h apiHandler) ListVALiveScheduleEnums(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.stacks.vaStaff.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.ListVALiveScheduleEnums(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListVALiveConnections(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.stacks.vaMember.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.ListConnections(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) DeleteVALiveConnection(w stdhttp.ResponseWriter, r *stdhttp.Request, organizationID openapi_types.UUID) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.DeleteConnection(w, r, organizationID.String())
	})).ServeHTTP(w, r)
}

func (h apiHandler) PingVALiveConnection(w stdhttp.ResponseWriter, r *stdhttp.Request, organizationID openapi_types.UUID) {
	h.stacks.vaStaff.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.PingConnection(w, r, organizationID.String())
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListVALiveOrganizationAircraft(w stdhttp.ResponseWriter, r *stdhttp.Request, organizationID openapi_types.UUID) {
	h.stacks.vaMember.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.ListOrganizationAircraft(w, r, organizationID)
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListVALiveOrganizationSchedules(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	organizationID openapi_types.UUID,
	params politburoapi.ListVALiveOrganizationSchedulesParams,
) {
	h.stacks.vaMember.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.ListOrganizationSchedules(w, r, organizationID, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListVALiveAircraftFlights(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	organizationID openapi_types.UUID,
	aircraftID openapi_types.UUID,
	params politburoapi.ListVALiveAircraftFlightsParams,
) {
	h.stacks.vaMember.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.ListAircraftFlights(w, r, organizationID, aircraftID, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListVALiveOrganizationFlights(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	organizationID openapi_types.UUID,
	params politburoapi.ListVALiveOrganizationFlightsParams,
) {
	h.stacks.vaMember.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.ListOrganizationFlights(w, r, organizationID, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) CreateVALiveAircraftSchedule(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	organizationID openapi_types.UUID,
	aircraftID openapi_types.UUID,
) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.CreateVALiveAircraftSchedule(w, r, organizationID, aircraftID)
	})).ServeHTTP(w, r)
}

func (h apiHandler) UpdateVALiveSchedule(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	organizationID openapi_types.UUID,
	scheduleID openapi_types.UUID,
) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.UpdateVALiveSchedule(w, r, organizationID, scheduleID)
	})).ServeHTTP(w, r)
}

func (h apiHandler) UpdateVALiveScheduleFlightPlan(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	organizationID openapi_types.UUID,
	scheduleID openapi_types.UUID,
) {
	h.stacks.vaAdmin.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.UpdateVALiveScheduleFlightPlan(w, r, organizationID, scheduleID)
	})).ServeHTTP(w, r)
}

func (h apiHandler) AssignVALiveScheduleToMe(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	organizationID openapi_types.UUID,
) {
	h.stacks.vaMember.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.AssignVALiveScheduleToMe(w, r, organizationID)
	})).ServeHTTP(w, r)
}

func (h apiHandler) ListVALiveScheduleBookings(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	params politburoapi.ListVALiveScheduleBookingsParams,
) {
	h.stacks.vaStaff.apply(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.handlers.liveConn.ListVALiveScheduleBookings(w, r, params)
	})).ServeHTTP(w, r)
}

func writeParameterError(w stdhttp.ResponseWriter, _ *stdhttp.Request, err error) {
	response.WriteError(w, stdhttp.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
}
