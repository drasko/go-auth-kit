package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/drasko/go-auth/auth"
	"github.com/drasko/go-auth/auth/api"
	"github.com/drasko/go-auth/bcrypt"
	"github.com/drasko/go-auth/jwt"
	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mainflux/mainflux/writer/cassandra"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	port           int    = 8089
	defPostgresURL string = "http://localhost:8180"
	envPostgresURL string = "AUTH_POSTGRES_URL"
)

type config struct {
	Port        int
	AuthURL     string
	PostgresURL string
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func main() {
	cfg := config{
		Port:       port,
		PotgresURL: getenv(envPostgresURL, defPotgresURL),
	}

	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	db, err := gorm.Open("postgres", "auth.db")
	if err != nil {
		logger.Log("error", err)
		os.Exit(1)
	}
	defer db.Close()

	users := postgres.NewUserRepository(session)
	hasher := bcrypt.NewHasher()
	idp := jwt.NewIdentityProvider(cfg.Secret)

	var svc auth.Service
	svc = auth.NewService(users, hasher, idp)
	svc = api.NewLoggingService(logger, svc)

	fields := []string{"method"}
	svc = api.NewMetricService(
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "auth",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, fields),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "auth",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fields),
		svc,
	)

	errs := make(chan error, 2)

	go func() {
		p := fmt.Sprintf(":%d", cfg.Port)
		errs <- http.ListenAndServe(p, api.MakeHandler(svc))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	logger.Log("terminated", <-errs)
}
