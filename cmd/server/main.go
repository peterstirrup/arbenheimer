package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/peterstirrup/arbenheimer/internal/domain/usecases"
	"github.com/peterstirrup/arbenheimer/internal/inbound/server"
	"github.com/peterstirrup/arbenheimer/internal/inbound/server/pb"
	"github.com/peterstirrup/arbenheimer/internal/outbound/redis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

const (
	maxConnectionIdle     = 10 * time.Minute
	maxConnectionAge      = 5 * time.Minute
	maxConnectionAgeGrace = 5 * time.Minute
	defaultTime           = 5 * time.Minute
)

type cliArgs struct {
	Host      string `arg:"--host,required,env:HOST"`
	LogLevel  string `arg:"--log-level,env:LOG_LEVEL" default:"debug"`
	Port      int    `arg:"env:PORT" default:"9000"`
	RedisHost string `arg:"--redis-host,required,env:REDIS_HOST"`
	RedisPort string `arg:"--redis-port,required,env:REDIS_PORT"`
}

func main() {
	var args cliArgs
	arg.MustParse(&args)

	logLevel, err := zerolog.ParseLevel(args.LogLevel)
	if err != nil {
		log.Warn().Msg("Failed to parse log level, defaulting to debug")
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	ctx := context.Background()

	rc := redis.NewClient(redis.Config{Host: args.RedisHost, Port: args.RedisPort})

	u := usecases.NewMarket(usecases.MarketConfig{
		Store:   rc,
		TimeNow: time.Now,
	})

	s := server.NewServer(server.Config{MarketUseCases: u})
	gs, err := newGRPCServer(ctx, args.Host, args.Port)

	pb.RegisterArbenheimerServiceServer(gs.s, s)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start gRPC server")
	}

	if err := gs.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to run gRPC server")
	}
}

// gRPCServer type wraps the base grpc.Server type and simplifies serving
// over TCP connections. The Run method provides context cancellation handling
// not provided by the base type.
type gRPCServer struct {
	s       *grpc.Server
	lis     net.Listener
	hs      *health.Server
	address string
}

// newGRPCServer returns a new gRPC server.
func newGRPCServer(ctx context.Context, host string, port int, opts ...grpc.ServerOption) (*gRPCServer, error) {
	address := fmt.Sprintf("%s:%d", host, port)

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("%w while starting tcp listener", err)
	}
	log.Debug().Msg("tcp listener started")

	c := defaultGRPCConfig()
	c.serverOptions = append(c.serverOptions, opts...)

	s := grpc.NewServer(c.serverOptions...)
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)

	return &gRPCServer{s: s, lis: lis, address: address, hs: healthServer}, nil
}

type grpcConfig struct {
	serverOptions []grpc.ServerOption
}

func defaultGRPCConfig() grpcConfig {
	return grpcConfig{
		serverOptions: []grpc.ServerOption{
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     maxConnectionIdle,
				MaxConnectionAge:      maxConnectionAge,
				MaxConnectionAgeGrace: maxConnectionAgeGrace,
				Time:                  defaultTime,
			}),
		},
	}
}

// Run starts the gRPC server and blocks until the context is cancelled.
func (s *gRPCServer) Run(ctx context.Context) error {
	log.Info().Msg("starting gRPC server")

	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			s.hs.SetServingStatus("ready", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			s.s.GracefulStop()
			<-done

		case <-done:
		}
	}()

	err := s.s.Serve(s.lis)
	done <- struct{}{}

	if err != nil {
		return err
	}

	return ctx.Err()
}
