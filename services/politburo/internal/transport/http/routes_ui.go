package http

import (
	"infinite-experiment/politburo/internal/app"
	"infinite-experiment/politburo/internal/routes"
	appmiddleware "infinite-experiment/politburo/internal/transport/http/middleware"
	uihttp "infinite-experiment/politburo/internal/transport/http/ui"

	"github.com/go-chi/chi/v5"
)

func mountUI(router chi.Router, h handlers, application *app.App) {
	router.Get(routes.AuthLogin, h.ui.Login)
	router.Get(routes.AuthLogout, h.ui.Logout)

	router.Group(func(ui chi.Router) {
		ui.Use(appmiddleware.UISessionAuth(application.Sessions))
		ui.Get(routes.Dashboard, h.ui.Dashboard)
		ui.Get(routes.Dashboard+"/", h.ui.Dashboard)
		ui.Get(routes.Admin, h.ui.AdminRedirect)
		ui.Get(routes.Admin+"/", h.ui.AdminRedirect)
		ui.Get(routes.AdminGeneral, h.ui.Admin)
		ui.Get(routes.AdminGeneral+"/", h.ui.Admin)
		ui.Get(routes.AdminLive, h.ui.Admin)
		ui.Get(routes.AdminLive+"/", h.ui.Admin)
		ui.Get(routes.Live, h.ui.LiveRedirect)
		ui.Get(routes.Live+"/", h.ui.LiveRedirect)
		ui.Get(routes.LiveAircraft, h.ui.LiveMode)
		ui.Get(routes.LiveAircraft+"/", h.ui.LiveMode)
		ui.Get(routes.LiveSchedules, h.ui.LiveMode)
		ui.Get(routes.LiveSchedules+"/", h.ui.LiveMode)
		ui.Get(routes.LiveFlights, h.ui.LiveMode)
		ui.Get(routes.LiveFlights+"/", h.ui.LiveMode)
		ui.Get(routes.OperatorReports, h.ui.OperatorReports)
		ui.Post(routes.OperatorReportsResolve, h.ui.ResolveOperatorReport)
	})

	router.Get(routes.MapsFlightsActive, h.ui.ActiveFlightsMap)
	router.Get(routes.MapsFlightsActiveServers, h.ui.MapServerOptions)
	router.Get(routes.MapsFlightsActiveFlights, h.ui.MapActiveFlightsPartial)
	router.Get(routes.MapsFlightsActiveFlightDetail, h.ui.MapFlightDetailPartial)
	router.Get(routes.Home, h.ui.Home)
	router.Get(routes.TermsConditions, h.ui.TermsConditions)
	router.Get(routes.Privacy, h.ui.PrivacyPolicy)
	router.Get(routes.Contact, h.ui.Contact)
	router.Post(routes.Contact, h.ui.SubmitContact)
	router.Handle(routes.StaticBrandPrefix+"*", uihttp.Brand())
	router.Handle(routes.StaticPrefix+"*", uihttp.Static())
	router.NotFound(h.ui.NotFound)
}
