package http

import (
	"infinite-experiment/politburo/internal/app"
	"infinite-experiment/politburo/internal/routes"
	appmiddleware "infinite-experiment/politburo/internal/transport/http/middleware"

	"github.com/go-chi/chi/v5"
)

func mountBot(router chi.Router, h handlers, application *app.App) {
	router.Group(func(bot chi.Router) {
		bot.Use(botStack(application)...)
		bot.Get(routes.UserStatus, h.identity.Status)
		bot.Post(routes.ReportsOccupiedIFC, h.identity.ReportOccupiedIFC)
	})

	router.Group(func(op chi.Router) {
		op.Use(appmiddleware.RequirePlatformOperator(application.Config.PlatformOperatorDiscordIDs))
		op.Get(routes.APIOperatorReports, h.identity.ListOperatorReports)
		op.Post(routes.APIOperatorReportsResolve, h.identity.ResolveOperatorReport)
	})
}
