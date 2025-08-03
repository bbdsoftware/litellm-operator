/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// trueValue is the string representation of true for boolean parsing
	trueValue = "true"
)

// OperatorConfig holds the global configuration for the operator
type OperatorConfig struct {
	// Default connection settings
	DefaultLitellmURL string
	DefaultMasterKey  string

	// Reconcile settings
	ReconcileTimeout        time.Duration
	ReconcileInterval       time.Duration
	MaxConcurrentReconciles int

	// Resource management
	DefaultNamespace string
	ResourcePrefix   string

	// Logging and monitoring
	LogLevel        string
	MetricsPort     int
	HealthProbePort int

	// Feature flags
	EnableLeaderElection bool
	EnableMetrics        bool
	EnableHealthChecks   bool

	// Advanced settings
	LeaderElectionID        string
	LeaderElectionNamespace string

	// ConfigMap settings
	ConfigMapName      string
	ConfigMapNamespace string
	ConfigMapKey       string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *OperatorConfig {
	return &OperatorConfig{
		DefaultLitellmURL:       "",
		DefaultMasterKey:        "",
		ReconcileTimeout:        30 * time.Second,
		ReconcileInterval:       10 * time.Second,
		MaxConcurrentReconciles: 1,
		DefaultNamespace:        "default",
		ResourcePrefix:          "litellm",
		LogLevel:                "info",
		MetricsPort:             8080,
		HealthProbePort:         8081,
		EnableLeaderElection:    true,
		EnableMetrics:           true,
		EnableHealthChecks:      true,
		LeaderElectionID:        "litellm-operator",
		LeaderElectionNamespace: "",
		ConfigMapName:           "litellm-operator-config",
		ConfigMapNamespace:      "",
		ConfigMapKey:            "config.yaml",
	}
}

// LoadFromFlags loads configuration from command line flags
func (c *OperatorConfig) LoadFromFlags() {
	// Default LiteLLM settings
	if url := os.Getenv("DEFAULT_LITELLM_URL"); url != "" {
		c.DefaultLitellmURL = url
	}
	if key := os.Getenv("DEFAULT_MASTER_KEY"); key != "" {
		c.DefaultMasterKey = key
	}

	// Reconcile settings
	if timeout := os.Getenv("RECONCILE_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			c.ReconcileTimeout = duration
		}
	}
	if interval := os.Getenv("RECONCILE_INTERVAL"); interval != "" {
		if duration, err := time.ParseDuration(interval); err == nil {
			c.ReconcileInterval = duration
		}
	}
	if maxReconciles := os.Getenv("MAX_CONCURRENT_RECONCILES"); maxReconciles != "" {
		if max, err := strconv.Atoi(maxReconciles); err == nil && max > 0 {
			c.MaxConcurrentReconciles = max
		}
	}

	// Resource management
	if namespace := os.Getenv("DEFAULT_NAMESPACE"); namespace != "" {
		c.DefaultNamespace = namespace
	}
	if prefix := os.Getenv("RESOURCE_PREFIX"); prefix != "" {
		c.ResourcePrefix = prefix
	}

	// Logging and monitoring
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.LogLevel = strings.ToLower(logLevel)
	}
	if metricsPort := os.Getenv("METRICS_PORT"); metricsPort != "" {
		if port, err := strconv.Atoi(metricsPort); err == nil && port > 0 {
			c.MetricsPort = port
		}
	}
	if healthPort := os.Getenv("HEALTH_PROBE_PORT"); healthPort != "" {
		if port, err := strconv.Atoi(healthPort); err == nil && port > 0 {
			c.HealthProbePort = port
		}
	}

	// Feature flags
	if enableLeaderElection := os.Getenv("ENABLE_LEADER_ELECTION"); enableLeaderElection != "" {
		c.EnableLeaderElection = strings.ToLower(enableLeaderElection) == trueValue
	}
	if enableMetrics := os.Getenv("ENABLE_METRICS"); enableMetrics != "" {
		c.EnableMetrics = strings.ToLower(enableMetrics) == trueValue
	}
	if enableHealthChecks := os.Getenv("ENABLE_HEALTH_CHECKS"); enableHealthChecks != "" {
		c.EnableHealthChecks = strings.ToLower(enableHealthChecks) == trueValue
	}

	// Advanced settings
	if leaderElectionID := os.Getenv("LEADER_ELECTION_ID"); leaderElectionID != "" {
		c.LeaderElectionID = leaderElectionID
	}
	if leaderElectionNamespace := os.Getenv("LEADER_ELECTION_NAMESPACE"); leaderElectionNamespace != "" {
		c.LeaderElectionNamespace = leaderElectionNamespace
	}

	// ConfigMap settings
	if configMapName := os.Getenv("CONFIG_MAP_NAME"); configMapName != "" {
		c.ConfigMapName = configMapName
	}
	if configMapNamespace := os.Getenv("CONFIG_MAP_NAMESPACE"); configMapNamespace != "" {
		c.ConfigMapNamespace = configMapNamespace
	}
	if configMapKey := os.Getenv("CONFIG_MAP_KEY"); configMapKey != "" {
		c.ConfigMapKey = configMapKey
	}
}

// LoadFromConfigMap loads configuration from a Kubernetes ConfigMap
func (c *OperatorConfig) LoadFromConfigMap(ctx context.Context, client client.Client, namespace string) error {
	log := log.FromContext(ctx)

	// Use provided namespace or default
	configNamespace := c.ConfigMapNamespace
	if configNamespace == "" {
		configNamespace = namespace
	}

	// Get the ConfigMap
	configMap := &corev1.ConfigMap{}
	configMapKey := types.NamespacedName{
		Name:      c.ConfigMapName,
		Namespace: configNamespace,
	}

	if err := client.Get(ctx, configMapKey, configMap); err != nil {
		if errors.IsNotFound(err) {
			log.Info("ConfigMap not found, using default configuration",
				"configMap", c.ConfigMapName,
				"namespace", configNamespace)
			return nil
		}
		return fmt.Errorf("failed to get ConfigMap %s: %w", c.ConfigMapName, err)
	}

	// Parse configuration from ConfigMap data
	if configData, exists := configMap.Data[c.ConfigMapKey]; exists {
		return c.parseConfigData(configData)
	}

	log.Info("ConfigMap key not found, using default configuration",
		"configMap", c.ConfigMapName,
		"key", c.ConfigMapKey)
	return nil
}

// parseConfigData parses configuration data from ConfigMap
func (c *OperatorConfig) parseConfigData(data string) error {
	// Simple key-value parsing for now
	// In a production environment, you might want to use YAML or JSON parsing
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "DEFAULT_LITELLM_URL":
			c.DefaultLitellmURL = value
		case "DEFAULT_MASTER_KEY":
			c.DefaultMasterKey = value
		case "RECONCILE_TIMEOUT":
			if duration, err := time.ParseDuration(value); err == nil {
				c.ReconcileTimeout = duration
			}
		case "RECONCILE_INTERVAL":
			if duration, err := time.ParseDuration(value); err == nil {
				c.ReconcileInterval = duration
			}
		case "MAX_CONCURRENT_RECONCILES":
			if max, err := strconv.Atoi(value); err == nil && max > 0 {
				c.MaxConcurrentReconciles = max
			}
		case "DEFAULT_NAMESPACE":
			c.DefaultNamespace = value
		case "RESOURCE_PREFIX":
			c.ResourcePrefix = value
		case "LOG_LEVEL":
			c.LogLevel = strings.ToLower(value)
		case "METRICS_PORT":
			if port, err := strconv.Atoi(value); err == nil && port > 0 {
				c.MetricsPort = port
			}
		case "HEALTH_PROBE_PORT":
			if port, err := strconv.Atoi(value); err == nil && port > 0 {
				c.HealthProbePort = port
			}
		case "ENABLE_LEADER_ELECTION":
			c.EnableLeaderElection = strings.ToLower(value) == trueValue
		case "ENABLE_METRICS":
			c.EnableMetrics = strings.ToLower(value) == trueValue
		case "ENABLE_HEALTH_CHECKS":
			c.EnableHealthChecks = strings.ToLower(value) == trueValue
		case "LEADER_ELECTION_ID":
			c.LeaderElectionID = value
		case "LEADER_ELECTION_NAMESPACE":
			c.LeaderElectionNamespace = value
		}
	}

	return nil
}

// Validate validates the configuration
func (c *OperatorConfig) Validate() error {
	if c.ReconcileTimeout <= 0 {
		return fmt.Errorf("reconcile timeout must be positive")
	}
	if c.ReconcileInterval <= 0 {
		return fmt.Errorf("reconcile interval must be positive")
	}
	if c.MaxConcurrentReconciles <= 0 {
		return fmt.Errorf("max concurrent reconciles must be positive")
	}
	if c.MetricsPort <= 0 || c.MetricsPort > 65535 {
		return fmt.Errorf("metrics port must be between 1 and 65535")
	}
	if c.HealthProbePort <= 0 || c.HealthProbePort > 65535 {
		return fmt.Errorf("health probe port must be between 1 and 65535")
	}
	if c.MetricsPort == c.HealthProbePort {
		return fmt.Errorf("metrics port and health probe port must be different")
	}

	return nil
}

// String returns a string representation of the configuration
func (c *OperatorConfig) String() string {
	return fmt.Sprintf(`
Operator Configuration:
  Default LiteLLM URL: %s
  Default Master Key: %s
  Reconcile Timeout: %v
  Reconcile Interval: %v
  Max Concurrent Reconciles: %d
  Default Namespace: %s
  Resource Prefix: %s
  Log Level: %s
  Metrics Port: %d
  Health Probe Port: %d
  Enable Leader Election: %t
  Enable Metrics: %t
  Enable Health Checks: %t
  Leader Election ID: %s
  Leader Election Namespace: %s
  ConfigMap Name: %s
  ConfigMap Namespace: %s
  ConfigMap Key: %s
`,
		c.DefaultLitellmURL,
		c.DefaultMasterKey,
		c.ReconcileTimeout,
		c.ReconcileInterval,
		c.MaxConcurrentReconciles,
		c.DefaultNamespace,
		c.ResourcePrefix,
		c.LogLevel,
		c.MetricsPort,
		c.HealthProbePort,
		c.EnableLeaderElection,
		c.EnableMetrics,
		c.EnableHealthChecks,
		c.LeaderElectionID,
		c.LeaderElectionNamespace,
		c.ConfigMapName,
		c.ConfigMapNamespace,
		c.ConfigMapKey,
	)
}
