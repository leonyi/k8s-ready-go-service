package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"
	"github.com/ardanlabs/service/api/services/api/debug"
	"github.com/ardanlabs/service/foundation/logger"
)

var tag = "develop"

func main() {
	// constructs a pointer to the logger.
	var log *logger.Logger

	events := logger.Events{
		// this should be done outside of this once we have a more ops environment. This is just to ensure we have something in place to test the events.
		Error: func(ctx context.Context, r logger.Record) {
			log.Info(ctx, "******* SEND ALERT *******")
		},
	}

	traceIDFn := func(ctx context.Context) string {
		return "" //web.GetTraceID(ctx)
	}

	// Construct the logger with a factory function
	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "SALES", traceIDFn, events)

	// -------------------------------------------------------------------------

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)
		os.Exit(1)
	}

}

func run(ctx context.Context, log *logger.Logger) error {
	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	cfg := struct {
		conf.Version
		Web struct {
			ReadTimeout        time.Duration `conf:"default:5s"`
			WriteTimeout       time.Duration `conf:"default:10s"`
			IdleTimeout        time.Duration `conf:"default:120s"`
			ShutdownTimeout    time.Duration `conf:"default:20s"`
			APIHost            string        `conf:"default:0.0.0.0:3000"`
			DebugHost          string        `conf:"default:0.0.0.0:3010"`
			CORSAllowedOrigins []string      `conf:"default:*,mask"`
		}
	}{
		Version: conf.Version{
			Build: tag,
			Desc:  "Sales",
		},
	}

	// Variable prefix is used to specify the prefix for environment variables when parsing the configuration.
	// In this case, the prefix is set to "SALES", which means that any environment variable that starts with "SALES_" will be considered when parsing the configuration.
	const prefix = "SALES"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		// Both help and version flags are handled by the configuration
		// library.  When either is requested the library returns a
		// non‑nil error (ErrHelpWanted or ErrVersionWanted) along with a
		// formatted string.  The original code only checked for help which
		// meant `--version` bubbled up as an error and caused the service to
		// exit with status 1.
		if errors.Is(err, conf.ErrHelpWanted) || errors.Is(err, conf.ErrVersionWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info(ctx, "starting service", "version", cfg.Build)
	defer log.Info(ctx, "shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Info(ctx, "startup", "config", out)

	expvar.NewString("build").Set(cfg.Build)

	// -------------------------------------------------------------------------
	// Start Debug Service

	go func() {
		log.Info(ctx, "startup", "status", "debug v1 router started", "host", cfg.Web.DebugHost)

		// http.DefaultServeMux is the default HTTP request multiplexer in Go.
		// It is a built-in type that implements the http.Handler. Any package can register handlers to this default multiplexer using the http.HandleFunc or http.Handle functions.
		// It's a security risk to use it in lines like this one, instead define your own!
		if err := http.ListenAndServe(cfg.Web.DebugHost, debug.Mux()); err != nil {
			log.Error(ctx, "shutdown", "status", "debug v1 router closed", "host", cfg.Web.DebugHost, "msg", err)
		}
	}()

	// -------------------------------------------------------------------------

	// Creates a channel of type os.Signal with a buffer of 1 to receive shutdown signals.
	// The signal.Notify function is used to register the channel to receive notifications for SIGINT and SIGTERM signals,
	// which are common signals used to indicate that the program should terminate gracefully.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	sig := <-shutdown
	log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
	defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

	return nil
}
