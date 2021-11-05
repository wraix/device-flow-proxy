package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	cache "github.com/patrickmn/go-cache"

	"github.com/wraix/device-flow-proxy/app"
	"github.com/wraix/device-flow-proxy/router"
	"github.com/wraix/device-flow-proxy/tracing"

	"github.com/charmixer/oas/exporter"

	"github.com/rs/zerolog/log"
)

type serveCmd struct {
	Tracing struct {
		Enabled  bool   `long:"trace-enable" description:"Enable tracing"`
		Url      string `long:"trace-provider-url" description:"Trace provider endpoint to use instead of default"`
		Provider string `long:"trace-provider" description:"Provider to use for tracing" choice:"jaeger" default:"jaeger"`
	}
	Public struct {
		Port   int    `short:"p" long:"port" description:"Port to serve app on" default:"8080"`
		Ip     string `short:"i" long:"ip" description:"IP to serve app on" default:"0.0.0.0"`
		Domain string `short:"d" long:"domain" description:"Domain to access app through" default:"127.0.0.1"`
	}
	Timeout struct {
		Write      int `long:"write-timeout" description:"Timeout in seconds for write" default:"10"`
		Read       int `long:"read-timeout" description:"Timeout in seconds for read" default:"5"`
		ReadHeader int `long:"read-header-timeout" description:"Timeout in seconds for read-header" default:"5"`
		Idle       int `long:"idle-timeout" description:"Timeout in seconds for idle" default:"10"`
		Grace      int `long:"grace-timeout" description:"Timeout in seconds before shutting down" default:"15"`
	}
	DeviceCodeGrant struct {
		BaseUrl               string `long:"dcg-base-url" description:"The base url for the code flow UI in the proxy" default:"https://localhost:8080"`
		AuthorizationEndpoint string `long:"dcg-authorization-endpoint" description:"The endpoint for the OAuth2 Provider Authorization endpoint" default:"https://localhost:4444/oauth2/auth"`
		TokenEndpoint         string `long:"dcg-token-endpoint" description:"The endpoint for the OAuth2 Provider Token endpoint" default:"https://localhost:4444/oauth2/token"`
		PollIntervalInSeconds int    `long:"dcg-poll-interval" description:"How often in seconds should clients poll to check if user logged in" default:"5"`
		ExpiresIn             int    `long:"dcg-expires-in" description:"Timeout in seconds for when generated code expires" default:"300"`
	}
	TLS struct {
		Cert struct {
			Path string
		}
		Key struct {
			Path string
		}
	}
}

func (cmd *serveCmd) initTracing() func() {
	if !cmd.Tracing.Enabled || cmd.Tracing.Provider == "" {
		log.Debug().Msgf("Tracing is disabled")
		return nil
	}

	var err error

	exporter := tracing.SetupNilExporter()
	if cmd.Tracing.Provider == "jaeger" {
		exporter, err = tracing.SetupJaegerExporter(cmd.Tracing.Url)
	}

	if err != nil {
		log.Error().Err(err).Msg("Unable to setup trace exporter")
		return nil
	}

	if exporter == nil {
		log.Debug().Msg("No exporter was setup for tracing")
		return nil
	}

	shutdownTracing, err := tracing.SetupTracing(exporter, Application.Name, Application.Environment, Application.Version)
	if err == nil {
		return shutdownTracing
	}

	// Deny by default
	log.Error().Err(err).Msg("Failed to setup tracer")
	return nil
}

func (cmd *serveCmd) Execute(args []string) error {
	app.Env.Ip = cmd.Public.Ip
	app.Env.Port = cmd.Public.Port
	app.Env.Domain = cmd.Public.Domain
	app.Env.Addr = fmt.Sprintf("%s:%d", app.Env.Ip, app.Env.Port)

	shutdown := cmd.initTracing()
	defer shutdown()

	router := router.NewRouter(Application.Name, Application.Description, Application.Version)

	oasModel := exporter.ToOasModel(
		router.OpenAPI,
		exporter.WithQueryTag("query"),
		exporter.WithHeaderTag("header"),
		exporter.WithCookieTag("cookie"),
	)
	app.Env.OpenAPI = oasModel

	// Use simple in memory cache - WARNING: use persistent storage cache like redis in production!
	// Create a cache with a default expiration time of 5 minutes, and which purges expired items every 10 minutes
	app.Env.BaseUrl = cmd.DeviceCodeGrant.BaseUrl
	app.Env.AuthorizationEndpoint = cmd.DeviceCodeGrant.AuthorizationEndpoint
	app.Env.TokenEndpoint = cmd.DeviceCodeGrant.TokenEndpoint
	app.Env.PollIntervalInSeconds = cmd.DeviceCodeGrant.PollIntervalInSeconds // 5
	app.Env.CacheDefaultExpiration = cmd.DeviceCodeGrant.ExpiresIn
	app.Env.CachePurgeExpired = 10
	app.Env.Cache = cache.New(time.Second*time.Duration(app.Env.CacheDefaultExpiration), time.Minute*time.Duration(app.Env.CachePurgeExpired))

	// 3x. server handler er (router resolve, chain, router(chain resolved)
	//https://github.com/julienschmidt/httprouter
	srv := &http.Server{
		Addr: app.Env.Addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout:      time.Second * time.Duration(cmd.Timeout.Write),
		ReadTimeout:       time.Second * time.Duration(cmd.Timeout.Read),
		ReadHeaderTimeout: time.Second * time.Duration(cmd.Timeout.ReadHeader),
		IdleTimeout:       time.Second * time.Duration(cmd.Timeout.Idle),
		Handler:           router.Handle(), // chain.Then(router), // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Info().Msg("Listening on " + app.Env.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Error().Err(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(cmd.Timeout.Idle))
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Info().Msg("shutting down")
	os.Exit(0)

	return nil
}
