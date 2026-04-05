package provision

import (
	"fmt"
	"time"
)

type localWindowsProvisionSessionRunner[Session any] struct {
	setup            func(string, ProvisionOptions) (Session, error)
	afterSetup       func(string, Session, ProvisionOptions) error
	buildArgs        func(string, Session, ProvisionOptions) ([]string, func() error, error)
	cleanup          func(string, Session, ProvisionOptions) error
	afterSetupError  func(error, error) error
	buildArgsError   func(error, error) error
	provisionResult  func(error, error, error) error
	ansibleLogPrefix string
	runTimeout       time.Duration
}

func runLocalWindowsProvisionSession[Session any](projectDir string, options ProvisionOptions, runner localWindowsProvisionSessionRunner[Session]) error {
	session, err := runner.setup(projectDir, options)
	if err != nil {
		return err
	}

	if runner.afterSetup != nil {
		if err := runner.afterSetup(projectDir, session, options); err != nil {
			cleanupErr := runner.cleanup(projectDir, session, options)
			return runner.afterSetupError(err, cleanupErr)
		}
	}

	args, argsCleanup, err := runner.buildArgs(projectDir, session, options)
	if err != nil {
		cleanupErr := runner.cleanup(projectDir, session, options)
		return runner.buildArgsError(err, cleanupErr)
	}

	runErr := runAnsibleProvisionCommandFunc(projectDir, args, runner.runTimeout, runner.ansibleLogPrefix)

	var argsCleanupErr error
	if argsCleanup != nil {
		argsCleanupErr = argsCleanup()
	}
	cleanupErr := runner.cleanup(projectDir, session, options)

	return runner.provisionResult(runErr, argsCleanupErr, cleanupErr)
}

func formatLocalWindowsProvisionStepError(message string, err error, cleanupErr error, cleanupLabel string) error {
	if cleanupErr != nil {
		return fmt.Errorf("%s: %w (also failed to restore secure %s state: %v)", message, err, cleanupLabel, cleanupErr)
	}

	return fmt.Errorf("%s: %w", message, err)
}

func formatLocalWindowsProvisionOutcome(protocolLabel string, cleanupLabel string, runErr error, argsCleanupErr error, cleanupErr error) error {
	if runErr != nil {
		if argsCleanupErr != nil && cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows via %s: %w (also failed to clean ansible temp files: %v; cleanup failed: %v)", protocolLabel, runErr, argsCleanupErr, cleanupErr)
		}
		if argsCleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows via %s: %w (also failed to clean ansible temp files: %v)", protocolLabel, runErr, argsCleanupErr)
		}
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows via %s: %w (also failed to restore secure %s state: %v)", protocolLabel, runErr, cleanupLabel, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for local host windows via %s: %w", protocolLabel, runErr)
	}
	if argsCleanupErr != nil && cleanupErr != nil {
		return fmt.Errorf("failed to clean ansible temp files: %w (also failed to restore secure %s state: %v)", argsCleanupErr, cleanupLabel, cleanupErr)
	}
	if argsCleanupErr != nil {
		return fmt.Errorf("failed to clean ansible temp files: %w", argsCleanupErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to restore secure %s state after local host windows provision: %w", cleanupLabel, cleanupErr)
	}

	return nil
}
