package common

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

// Embedded eigenx version from release
var embeddedEigenXReleaseVersion = "Development"

// WithShutdown creates a new context that will be cancelled on SIGTERM/SIGINT
func WithShutdown(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		signal.Stop(sigChan)
		cancel()
		_, _ = fmt.Fprintln(os.Stderr, "caught interrupt, shutting down gracefully.")
	}()

	return ctx
}

type appEnvironmentContextKey struct{}

type AppEnvironment struct {
	CLIVersion string
	OS         string
	Arch       string
	UserUUID   string
}

func NewAppEnvironment(os, arch, userUuid string) *AppEnvironment {
	return &AppEnvironment{
		CLIVersion: embeddedEigenXReleaseVersion,
		OS:         os,
		Arch:       arch,
		UserUUID:   userUuid,
	}
}

func WithAppEnvironment(ctx *cli.Context) {
	user := getUserUUIDFromGlobalConfig()
	if user == "" {
		user = uuid.New().String()
	}

	ctx.Context = withAppEnvironment(ctx.Context, NewAppEnvironment(
		runtime.GOOS,
		runtime.GOARCH,
		user,
	))
}

func withAppEnvironment(ctx context.Context, appEnvironment *AppEnvironment) context.Context {
	return context.WithValue(ctx, appEnvironmentContextKey{}, appEnvironment)
}

func AppEnvironmentFromContext(ctx context.Context) (*AppEnvironment, bool) {
	env, ok := ctx.Value(appEnvironmentContextKey{}).(*AppEnvironment)
	return env, ok
}
