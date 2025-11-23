package build

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type RunProcessConfig struct {
	ExecutablePath     string
	Args               []string
	WorkingDir         string
	Timeout            time.Duration
	DelayBeforeStart   time.Duration
	Context            context.Context
	FailOnError        bool
	Retries            int
	RetryInterval      time.Duration
	InterruptRetryChan chan bool
}

func RunExternalProcess(config RunProcessConfig) context.Context {
	var ctx context.Context
	var cancel context.CancelFunc
	if config.Context != nil {
		ctx, cancel = context.WithTimeout(config.Context, config.Timeout)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), config.Timeout)
	}
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	if config.DelayBeforeStart > 0 {
		log.Printf("Delaying start of process %s by %s", config.ExecutablePath, config.DelayBeforeStart)
		select {
		case <-time.After(config.DelayBeforeStart):
			// Reset DelayBeforeStart to 0 after the delay
			config.DelayBeforeStart = 0
			// continue
		case sig := <-sigs:
			log.Printf("Process %s start interrupted by signal: %v", config.ExecutablePath, sig)
			return ctx
		case <-ctx.Done():
			log.Printf("Process %s start cancelled due to timeout or interruption: %v", config.ExecutablePath, ctx.Err())
			return ctx
		}
	}

	log.Printf("Starting process: %s %s", config.ExecutablePath, strings.Join(config.Args, " "))

	cmd := exec.CommandContext(ctx, config.ExecutablePath, config.Args...)
	cmd.Dir = config.WorkingDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to get stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("stdout:%s:  %s", config.ExecutablePath, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("stderr:%s:  %s", config.ExecutablePath, scanner.Text())
		}
	}()

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			if config.FailOnError {
				log.Fatalf("Process %s failed: %v", config.ExecutablePath, err)
			}
			log.Printf("Process %s finished with error: %v", config.ExecutablePath, err)
			return ctx
		}
		log.Printf("Process %s finished successfully.", config.ExecutablePath)
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		log.Printf("Process %s terminated due to timeout or interruption: %v", config.ExecutablePath, ctx.Err())
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		log.Printf("Process %s terminated due to signal: %v", config.ExecutablePath, sig)
	}
	return ctx
}

func RunExternalProcessWithRetries(config RunProcessConfig) context.Context {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= config.Retries && time.Since(startTime) < config.Timeout; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying process %s (attempt %d/%d) after error: %v", config.ExecutablePath, attempt, config.Retries, lastErr)
			time.Sleep(config.RetryInterval)
		}
		ctx := RunExternalProcess(config)
		if ctx.Err() == nil {
			return ctx
		}
		lastErr = ctx.Err()

		// Check if we received an interrupt signal to stop retries
		if config.InterruptRetryChan != nil {
			select {
			case <-config.InterruptRetryChan:
				log.Printf("Received interrupt signal, stopping retries for process %s", config.ExecutablePath)
				return ctx
			default:
				// No interrupt signal, continue
			}
		}
	}
	log.Printf("Process %s failed after %d attempts: %v", config.ExecutablePath, config.Retries+1, lastErr)
	return context.Background()
}

func RunBashScript(config RunProcessConfig) {
	scriptPath := config.ExecutablePath
	config.ExecutablePath = "bash"
	config.Args = append([]string{scriptPath}, config.Args...)
	RunExternalProcess(config)
}

func RunPowerShellScript(config RunProcessConfig) {
	scriptPath := config.ExecutablePath
	config.ExecutablePath = "powershell"
	config.Args = append([]string{"-File", scriptPath}, config.Args...)
	RunExternalProcess(config)
}

func RunCliCommand(workdir string, command string, args []string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workdir
	log.Printf("Running command: %s %s", command, strings.Join(args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to run command %s: %v", command, err)
	}
	log.Printf("Command output: %s", string(output))
	return output, err
}
