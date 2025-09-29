package testutils

import (
	"bytes"
	"os"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"

	"github.com/urfave/cli/v2"
)

// CreateTestAppWithNoopLoggerAndAccess creates a CLI app with no-op logger and returns both app and logger
func CreateTestAppWithNoopLoggerAndAccess(name string, flags []cli.Flag, action cli.ActionFunc) (*cli.App, *logger.NoopLogger) {
	noopLogger := logger.NewNoopLogger()
	noopProgressTracker := logger.NewNoopProgressTracker()
	app := &cli.App{
		Name:  name,
		Flags: flags,
		Before: func(cCtx *cli.Context) error {
			// Use the same logger instance
			ctx := common.WithLogger(cCtx.Context, noopLogger)
			ctx = common.WithProgressTracker(ctx, noopProgressTracker)
			cCtx.Context = ctx
			return nil
		},
		Action: action,
	}
	return app, noopLogger
}

func CaptureOutput(fn func()) (stdout string, stderr string) {
	// Get the logger
	log, _ := common.GetLogger(true)
	// Capture stdout
	origStdout := os.Stdout
	origStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rOut); err != nil {
			log.Warn("failed to read stdout: %v", err)
		}
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rErr); err != nil {
			log.Warn("failed to read stdout: %v", err)
		}
		errC <- buf.String()
	}()

	// Run target code
	fn()

	// Restore
	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	stdout = <-outC
	stderr = <-errC

	return stdout, stderr
}
