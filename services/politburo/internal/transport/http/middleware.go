package http

import (
	stdhttp "net/http"

	"infinite-experiment/politburo/internal/app"
	appmiddleware "infinite-experiment/politburo/internal/transport/http/middleware"
)

type middlewareStack []func(stdhttp.Handler) stdhttp.Handler

func botStack(application *app.App) middlewareStack {
	return middlewareStack{
		appmiddleware.RequireDiscordBotContext(),
		appmiddleware.EnrichDiscordMembership(application.Resolver),
		appmiddleware.RateLimit(application.Cache, appmiddleware.RateLimitGroupRegistration, appmiddleware.RateLimitRegistration),
	}
}

func godStack(application *app.App) middlewareStack {
	return middlewareStack{
		appmiddleware.RequireDiscordBotContext(),
		appmiddleware.RequirePlatformOperator(application.Config.PlatformOperatorDiscordIDs),
	}
}

func vaAdminStack(application *app.App) middlewareStack {
	return middlewareStack{
		appmiddleware.EnrichDiscordMembership(application.Resolver),
		appmiddleware.RequireAdmin(),
	}
}

func currentUserStack(application *app.App) middlewareStack {
	return middlewareStack{
		appmiddleware.EnrichDiscordMembership(application.Resolver),
	}
}

func (s middlewareStack) apply(handler stdhttp.Handler) stdhttp.Handler {
	for i := len(s) - 1; i >= 0; i-- {
		handler = s[i](handler)
	}
	return handler
}
