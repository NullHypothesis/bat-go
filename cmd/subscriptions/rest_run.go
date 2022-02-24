package subscriptions

import (
	"context"
	"net/http"
	"time"

	// pprof imports
	_ "net/http/pprof"

	"github.com/brave-intl/bat-go/cmd"
	"github.com/brave-intl/bat-go/middleware"
	"github.com/brave-intl/bat-go/subscriptions"
	appctx "github.com/brave-intl/bat-go/utils/context"
	sentry "github.com/getsentry/sentry-go"
	"github.com/go-chi/chi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RestRun - Main entrypoint of the REST subcommand
// This function takes a cobra command and starts up the
// subscriptions rest microservice.
func RestRun(command *cobra.Command, args []string) {
	ctx := command.Context()
	logger, err := appctx.GetLogger(ctx)
	cmd.Must(err)
	// add profiling flag to enable profiling routes
	if viper.GetString("pprof-enabled") != "" {
		// pprof attaches routes to default serve mux
		// host:6061/debug/pprof/
		go func() {
			logger.Error().Err(http.ListenAndServe(":6061", http.DefaultServeMux))
		}()
	}

	// add our command line params to context
	ctx = context.WithValue(ctx, appctx.PaymentServerCTXKey, viper.Get("payments-service"))

	// setup the service now
	s, err := subscriptions.InitService(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initalize subscriptions service")
	}

	// do rest endpoints
	r := cmd.SetupRouter(command.Context())
	r.Get("/v1/rooms/{name}", middleware.InstrumentHandlerFunc("PostCreateRoomHandler", subscriptions.PostCreateRoomHandler(s)).ServeHTTP)

	// make sure exceptions go to sentry
	defer sentry.Flush(time.Second * 2)

	go func() {
		err := http.ListenAndServe(":9090", middleware.Metrics())
		if err != nil {
			sentry.CaptureException(err)
			logger.Panic().Err(err).Msg("metrics HTTP server start failed!")
		}
	}()

	// setup server, and run
	srv := http.Server{
		Addr:         viper.GetString("address"),
		Handler:      chi.ServerBaseContext(ctx, r),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	if err = srv.ListenAndServe(); err != nil {
		sentry.CaptureException(err)
		logger.Fatal().Err(err).Msg("HTTP server start failed!")
	}
}
