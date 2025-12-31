package virtualkey

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	LitellmImage = "ghcr.io/berriai/litellm-database:main-v1.74.9.rc.1"
)

// LitellmDockerFixture manages the lifecycle of a LiteLLM and Postgres docker container pair for testing
type LitellmDockerFixture struct {
	NetworkName  string
	PostgresName string
	LitellmName  string
	MasterKey    string
	BaseURL      string
	Port         string
}

// NewLitellmDockerFixture creates a new fixture instance with unique names
func NewLitellmDockerFixture() *LitellmDockerFixture {
	id := uuid.New().String()[:8]
	return &LitellmDockerFixture{
		NetworkName:  "litellm-net-" + id,
		PostgresName: "postgres-" + id,
		LitellmName:  "litellm-" + id,
		MasterKey:    "sk-" + uuid.New().String(),
	}
}

// Setup starts the Postgres and LiteLLM containers and waits for them to be ready
func (f *LitellmDockerFixture) Setup(ctx context.Context) error {
	// Create network
	if err := runCommand("docker", "network", "create", f.NetworkName); err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// Start Postgres
	if err := runCommand("docker", "run", "-d", "--name", f.PostgresName, "--network", f.NetworkName, "-e", "POSTGRES_PASSWORD=password", "postgres:latest"); err != nil {
		f.Teardown() // Cleanup if postgres fails
		return fmt.Errorf("failed to start postgres: %w", err)
	}

	// Wait a bit for Postgres to initialize
	time.Sleep(5 * time.Second)

	image := LitellmImage
	if os.Getenv("LITELLM_IMAGE") != "" {
		image = os.Getenv("LITELLM_IMAGE")
	}

	// Start LiteLLM
	// We map container port 4000 to a random host port using "-p 0:4000"
	cmd := exec.Command("docker", "run", "-d", "--name", f.LitellmName, "--network", f.NetworkName, "-p", "0:4000",
		"-e", "DATABASE_URL=postgresql://postgres:password@"+f.PostgresName+":5432/postgres",
		"-e", "LITELLM_MASTER_KEY="+f.MasterKey,
		image,
		"--port", "4000",
		"--model", "huggingface/bigcode/starcoder")

	if out, err := cmd.CombinedOutput(); err != nil {
		f.Teardown()
		return fmt.Errorf("failed to start litellm: %s, %w", out, err)
	}

	// Get assigned port
	// Output format example: 0.0.0.0:32768
	out, err := exec.Command("docker", "port", f.LitellmName, "4000").CombinedOutput()
	if err != nil {
		f.Teardown()
		return fmt.Errorf("failed to get port: %w", err)
	}

	// Parse port
	outputStr := strings.TrimSpace(string(out))
	parts := strings.Split(outputStr, ":")
	if len(parts) < 2 {
		f.Teardown()
		return fmt.Errorf("invalid port output: %s", outputStr)
	}
	f.Port = parts[len(parts)-1]
	f.BaseURL = "http://localhost:" + f.Port

	// Wait for readiness
	if err := f.waitForReady(ctx); err != nil {
		f.Teardown()
		return err
	}

	return nil
}

// waitForReady polls the LiteLLM health endpoint until it returns 200 OK
func (f *LitellmDockerFixture) waitForReady(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.After(28 * time.Second) // slightly shorter than default test timeout

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for litellm to be ready at %s", f.BaseURL)
		case <-ticker.C:
			// Try /health/liveness first
			resp, err := client.Get(f.BaseURL + "/health/liveness")
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}

			// Fallback to root / if health check fails (older versions?)
			resp, err = client.Get(f.BaseURL + "/")
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// Teardown stops and removes the containers and network
func (f *LitellmDockerFixture) Teardown() {
	// Ignore errors during teardown, just try to clean up everything
	exec.Command("docker", "rm", "-f", f.LitellmName).Run()
	exec.Command("docker", "rm", "-f", f.PostgresName).Run()
	exec.Command("docker", "network", "rm", f.NetworkName).Run()
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("command %s %v failed: %s, %w", name, args, out, err)
	}
	return nil
}
