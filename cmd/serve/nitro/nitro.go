package nitro

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brave-intl/bat-go/cmd"
	"github.com/brave-intl/bat-go/middleware"
	"github.com/brave-intl/bat-go/payments"
	appctx "github.com/brave-intl/bat-go/utils/context"
	"github.com/brave-intl/bat-go/utils/handlers"
	"github.com/brave-intl/bat-go/utils/logging"
	"github.com/brave-intl/bat-go/utils/nitro"
	"github.com/go-chi/chi"
	chiware "github.com/go-chi/chi/middleware"
	"github.com/mdlayher/vsock"
	"github.com/rs/zerolog/hlog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// upstream-url - sets the upstream-url of the server to be started
	NitroServeCmd.PersistentFlags().String("upstream-url", "", "the upstream url to proxy requests to")
	// egress-address - sets the vosck address for the open proxy to listen on for outgoing traffic
	NitroServeCmd.PersistentFlags().String("egress-address", "", "vsock address for open proxy to bind on")
	// log-address - sets the vosck address for the log server to listen on
	NitroServeCmd.PersistentFlags().String("log-address", "", "vsock address for log server to bind on")

	cmd.Must(NitroServeCmd.MarkPersistentFlagRequired("upstream-url"))
	cmd.Must(viper.BindPFlag("upstream-url", NitroServeCmd.PersistentFlags().Lookup("upstream-url")))
	cmd.Must(viper.BindEnv("upstream-url", "UPSTREAM_URL"))
	cmd.Must(NitroServeCmd.MarkPersistentFlagRequired("egress-address"))
	cmd.Must(viper.BindPFlag("egress-address", NitroServeCmd.PersistentFlags().Lookup("egress-address")))
	cmd.Must(viper.BindEnv("egress-address", "EGRESS_ADDRESS"))
	cmd.Must(NitroServeCmd.MarkPersistentFlagRequired("log-address"))
	cmd.Must(viper.BindPFlag("log-address", NitroServeCmd.PersistentFlags().Lookup("log-address")))
	cmd.Must(viper.BindEnv("log-address", "LOG_ADDRESS"))

	NitroServeCmd.AddCommand(OutsideNitroServeCmd)
	NitroServeCmd.AddCommand(InsideNitroServeCmd)
	cmd.ServeCmd.AddCommand(NitroServeCmd)
}

// OutsideNitroServeCmd the nitro serve command
var OutsideNitroServeCmd = &cobra.Command{
	Use:   "outside-enclave",
	Short: "subcommand to serve a nitro micro-service",
	Run:   cmd.Perform("outside-enclave", RunNitroServerOutsideEnclave),
}

// InsideNitroServeCmd the nitro serve command
var InsideNitroServeCmd = &cobra.Command{
	Use:   "inside-enclave",
	Short: "subcommand to serve a nitro micro-service",
	Run:   cmd.Perform("inside-enclave", RunNitroServerInEnclave),
}

// NitroServeCmd the nitro serve command
var NitroServeCmd = &cobra.Command{
	Use:   "nitro",
	Short: "subcommand to serve a nitro micro-service",
}

// RunNitroServerInEnclave - start up the nitro server living inside the enclave
func RunNitroServerInEnclave(cmd *cobra.Command, args []string) error {
	fmt.Println("running inside encalve")
	ctx := cmd.Context()

	logaddr := viper.GetString("log-address")
	writer := nitro.NewVsockWriter(logaddr)
	ctx = context.WithValue(ctx, appctx.LogWriterKey, writer)
	ctx = context.WithValue(ctx, appctx.EgressProxyAddrCTXKey, viper.GetString("egress-address"))
	// special logger with writer
	ctx, logger := logging.SetupLogger(ctx)
	// setup router
	ctx, r := setupRouter(ctx)
	// setup the service now
	ctx, s, err := payments.NewService(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initalize payments service")
	}
	r.Use(s.ConfigurationMiddleware())
	// setup payments routes
	// prepare inserts transactions into qldb, returning a document which needs to be submitted by an authorizer
	r.Post("/v1/payments/prepare", middleware.InstrumentHandler("PrepareHandler", payments.PrepareHandler(s)).ServeHTTP)
	// submit will have an http signature from a known list of public keys
	r.Post("/v1/payments/submit", middleware.InstrumentHandler("SubmitHandler", s.AuthorizerSignedMiddleware()(payments.SubmitHandler(s))).ServeHTTP)
	// status to get the status and submission results from the submit
	r.Post("/v1/payments/{documentID}/status", middleware.InstrumentHandler("StatusHandler", payments.SubmitHandler(s)).ServeHTTP)

	// get the public key
	r.Get("/v1/configuration", handlers.AppHandler(payments.GetConfigurationHandler(s)).ServeHTTP)
	r.Patch("/v1/configuration", handlers.AppHandler(payments.PatchConfigurationHandler(s)).ServeHTTP)

	// setup listener
	addr := viper.GetString("address")
	port, err := strconv.Atoi(strings.Split(addr, ":")[1])
	if err != nil || uint32(port) > ^uint32(0) {
		// panic if there is an error, or if the port is too large to fit in uint32
		logger.Panic().Err(err).Msg("invalid --address")
	}

	// setup vsock listener
	l, err := vsock.Listen(uint32(port))
	if err != nil {
		logger.Panic().Err(err).Msg("listening on vsock port failed")
	}
	// setup server
	srv := http.Server{
		Handler:      chi.ServerBaseContext(ctx, r),
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 20 * time.Second,
	}
	// run the server in another routine
	logger.Fatal().Err(srv.Serve(l)).Msg("server shutdown")
	return nil
}

func setupRouter(ctx context.Context) (context.Context, *chi.Mux) {
	// base service logger
	logger := logging.Logger(ctx, "payments")
	// base router
	r := chi.NewRouter()
	// middlewares
	r.Use(chiware.RequestID)
	r.Use(middleware.RequestIDTransfer)
	r.Use(hlog.NewHandler(*logger))
	r.Use(hlog.UserAgentHandler("user_agent"))
	r.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	r.Use(middleware.RequestLogger(logger))
	r.Use(chiware.Timeout(15 * time.Second))
	// routes
	r.Method("GET", "/health-check", http.HandlerFunc(nitro.EnclaveHealthCheck))
	return ctx, r
}

// RunNitroServerOutsideEnclave - start up all the services which are outside
func RunNitroServerOutsideEnclave(cmd *cobra.Command, args []string) error {
	fmt.Println("running outside encalve")
	ctx := cmd.Context()
	logger, err := appctx.GetLogger(ctx)
	if err != nil {
		return err
	}

	egressaddr := strings.Split(viper.GetString("egress-address"), ":")
	fmt.Println(egressaddr)
	if len(egressaddr) != 2 {
		return fmt.Errorf("address must include port")
	}
	egressport, err := strconv.Atoi(egressaddr[1])
	if err != nil || egressport < 0 {
		return fmt.Errorf("port must be a valid uint32: %v", err)
	}

	server, err := nitro.NewReverseProxyServer(
		viper.GetString("address"),
		viper.GetString("upstream-url"),
	)
	if err != nil {
		return err
	}

	logaddr := strings.Split(viper.GetString("log-address"), ":")
	if len(logaddr) != 2 {
		return fmt.Errorf("address must include port")
	}
	logport, err := strconv.ParseUint(logaddr[1], 10, 32)
	if err != nil {
		return fmt.Errorf("port must be a valid uint32: %v", err)
	}
	logserve := nitro.NewVsockLogServer(ctx, uint32(logport))

	logger.Info().
		Str("version", ctx.Value(appctx.VersionCTXKey).(string)).
		Str("commit", ctx.Value(appctx.CommitCTXKey).(string)).
		Str("build_time", ctx.Value(appctx.BuildTimeCTXKey).(string)).
		Str("upstream-url", viper.GetString("upstream-url")).
		Str("address", viper.GetString("address")).
		Str("environment", viper.GetString("environment")).
		Msg("server starting")

	go logger.Error().Err(logserve.Serve(nil)).Msg("failed to start log server")

	go logger.Error().Err(nitro.ServeOpenProxy(
		ctx,
		uint32(egressport),
		10*time.Second,
	)).Msg("failed to start proxy server")

	return server.ListenAndServe()
}
