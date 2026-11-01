package http

import (
	"infinite-experiment/politburo/internal/app"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/transport/http/api/gameflights"
	"infinite-experiment/politburo/internal/transport/http/api/gamesessions"
	"infinite-experiment/politburo/internal/transport/http/api/health"
	"infinite-experiment/politburo/internal/transport/http/api/identity"
	liveapi "infinite-experiment/politburo/internal/transport/http/api/live"
	oauthapi "infinite-experiment/politburo/internal/transport/http/api/oauth"
	"infinite-experiment/politburo/internal/transport/http/api/signedlink"
	userapi "infinite-experiment/politburo/internal/transport/http/api/user"
	vaapi "infinite-experiment/politburo/internal/transport/http/api/va"
	uihttp "infinite-experiment/politburo/internal/transport/http/ui"
)

type handlers struct {
	health     *health.Handler
	sessions   *gamesessions.Handler
	flights    *gameflights.Handler
	signedLink *signedlink.Handler
	identity   *identity.Handler
	va         *vaapi.Handler
	user       *userapi.Handler
	oauth      *oauthapi.Handler
	live       *liveapi.Handler
	liveConn   *liveapi.ConnectionsHandler
	ui         *uihttp.Handler
}

func newHandlers(application *app.App) handlers {
	userHandler := userapi.NewHandler(application.Users)
	return handlers{
		health:   health.NewHandler(application.DB, application.Cache, application.Config.Jobs.Enabled, application.StartedAt),
		sessions: gamesessions.NewHandler(application.SessionsReader),
		flights:  gameflights.NewHandler(application.FlightsReader, application.Metrics),
		signedLink: signedlink.NewHandler(
			application.Users,
			application.Tickets,
			application.Resolver,
			application.VALookup,
			&virtualairlines.MapLink{VA: application.VirtualAirlines, Config: application.VAConfigs},
			application.Config.Auth.UIBaseURL,
		),
		identity: identity.NewHandler(
			application.Registration, application.Membership, application.VAInit, application.Status,
			application.Reports, application.Operator, application.VirtualAirlines,
		),
		va:    vaapi.NewHandler(application.VirtualAirlines, application.VAConfigs, application.Users, application.VAWebhookJob),
		user:  userHandler,
		oauth: oauthapi.NewHandler(
			application.IFOAuth,
			application.Config.OAuthRedirectURI(),
			application.Cache,
			userHandler,
			application.IFLive,
			application.VALiveTokens,
			application.VALiveConnections,
		),
		live: liveapi.NewHandler(application.IFLive, application.Cache, userHandler),
		liveConn: liveapi.NewConnectionsHandler(
			application.VALiveConnections,
			application.IFLive,
			application.VALiveTokens,
			userHandler,
		),
		ui: uihttp.NewHandler(
			application.UI, application.Sessions, application.Tickets, application.Reports, application.Contact, application.Cache,
			application.Operator, application.Config.PlatformOperatorDiscordIDs, application.SessionsReader,
			application.FlightsReader,
			application.IFOAuth != nil,
			application.VALiveGrants,
			&uihttp.LiveFleetDeps{
				Live:        application.IFLive,
				Tokens:      application.VALiveTokens,
				Connections: application.VALiveConnections,
				Liveries:    application.LiveryLookup,
			},
		),
	}
}
