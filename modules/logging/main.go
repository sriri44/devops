package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	dockerComposeFile = "docker-compose.logging.yml"
	lokiConfigFile    = "loki-config.yml"
	promtailConfigFile = "promtail-config.yml"
	grafanaDataSourceFile = "grafana-provisioning/datasources/datasource.yml"
	grafanaDashboardFile = "grafana-provisioning/dashboards/dashboard.yml"
	nextjsDashboardFile = "grafana-provisioning/dashboards/nextjs-logs.json"
)

type LoggingStackManager struct {
	workingDir string
}

func NewLoggingStackManager() *LoggingStackManager {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get working directory:", err)
	}
	return &LoggingStackManager{workingDir: wd}
}

func (lsm *LoggingStackManager) CreateDirectories() error {
	fmt.Println("📁 Creating necessary directories...")
	
	dirs := []string{
		"grafana-provisioning/datasources",
		"grafana-provisioning/dashboards",
		"/logs", // This might need sudo
	}
	
	for _, dir := range dirs {
		if strings.HasPrefix(dir, "/") {
			// Create system directory with sudo
			cmd := exec.Command("sudo", "mkdir", "-p", dir)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", dir, err)
			}
		} else {
			// Create local directory
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", dir, err)
			}
		}
	}
	
	fmt.Println("✅ Directories created successfully")
	return nil
}

func (lsm *LoggingStackManager) CreateDockerComposeFile() error {
	fmt.Println("🐳 Creating Docker Compose file...")
	
	content := `version: '3.8'

services:
  # Loki - Log aggregation system
  loki:
    image: grafana/loki:2.9.0
    container_name: loki
    ports:
      - "3100:3100"
    volumes:
      - ./loki-config.yml:/etc/loki/local-config.yaml
      - loki-data:/loki
    command: -config.file=/etc/loki/local-config.yaml
    networks:
      - logging
    restart: unless-stopped

  # Promtail - Log shipper
  promtail:
    image: grafana/promtail:2.9.0
    container_name: promtail
    volumes:
      - ./promtail-config.yml:/etc/promtail/config.yml
      - /logs:/logs:ro
      - /var/log:/var/log:ro
    command: -config.file=/etc/promtail/config.yml
    depends_on:
      - loki
    networks:
      - logging
    restart: unless-stopped

  # Grafana - Visualization and dashboards
  grafana:
    image: grafana/grafana:10.1.0
    container_name: grafana
    ports:
      - "3001:3000"
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana-provisioning:/etc/grafana/provisioning
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_USERS_ALLOW_SIGN_UP=false
    depends_on:
      - loki
    networks:
      - logging
    restart: unless-stopped

volumes:
  loki-data:
  grafana-data:

networks:
  logging:
    driver: bridge`

	return ioutil.WriteFile(dockerComposeFile, []byte(content), 0644)
}

func (lsm *LoggingStackManager) CreateLokiConfig() error {
	fmt.Println("📋 Creating Loki configuration...")
	
	content := `auth_enabled: false

server:
  http_listen_port: 3100
  grpc_listen_port: 9096

common:
  path_prefix: /loki
  storage:
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules
  replication_factor: 1
  ring:
    instance_addr: 127.0.0.1
    kvstore:
      store: inmemory

query_range:
  results_cache:
    cache:
      embedded_cache:
        enabled: true
        max_size_mb: 100

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

ruler:
  alertmanager_url: http://localhost:9093`

	return ioutil.WriteFile(lokiConfigFile, []byte(content), 0644)
}

func (lsm *LoggingStackManager) CreatePromtailConfig() error {
	fmt.Println("📋 Creating Promtail configuration...")
	
	content := `server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  # Your Next.js app logs
  - job_name: nextjs-app
    static_configs:
      - targets:
          - localhost
        labels:
          job: nextjs-app
          __path__: /logs/app.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            message: message
            timestamp: timestamp
      - labels:
          level:
      - timestamp:
          source: timestamp
          format: RFC3339

  # Docker container logs
  - job_name: docker-containers
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s
    relabel_configs:
      - source_labels: ['__meta_docker_container_name']
        regex: '/instaroid-container'
        action: keep
      - source_labels: ['__meta_docker_container_name']
        target_label: container_name
      - source_labels: ['__meta_docker_container_log_stream']
        target_label: stream`

	return ioutil.WriteFile(promtailConfigFile, []byte(content), 0644)
}

func (lsm *LoggingStackManager) CreateGrafanaDataSource() error {
	fmt.Println("📋 Creating Grafana datasource configuration...")
	
	content := `apiVersion: 1

datasources:
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    isDefault: true
    editable: true`

	return ioutil.WriteFile(grafanaDataSourceFile, []byte(content), 0644)
}

func (lsm *LoggingStackManager) CreateGrafanaDashboardConfig() error {
	fmt.Println("📋 Creating Grafana dashboard configuration...")
	
	content := `apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards`

	return ioutil.WriteFile(grafanaDashboardFile, []byte(content), 0644)
}

func (lsm *LoggingStackManager) CreateNextJSDashboard() error {
	fmt.Println("📊 Creating Next.js dashboard...")
	
	content := `{
  "dashboard": {
    "id": null,
    "title": "Next.js Application Logs",
    "tags": ["nextjs", "logs"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Log Volume (Last 5 minutes)",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(count_over_time({job=\"nextjs-app\"}[5m]))",
            "refId": "A"
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 0,
          "y": 0
        }
      },
      {
        "id": 2,
        "title": "Error Logs",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(count_over_time({job=\"nextjs-app\", level=\"error\"}[5m]))",
            "refId": "A"
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 12,
          "y": 0
        }
      },
      {
        "id": 3,
        "title": "Recent Logs",
        "type": "logs",
        "targets": [
          {
            "expr": "{job=\"nextjs-app\"}",
            "refId": "A"
          }
        ],
        "gridPos": {
          "h": 12,
          "w": 24,
          "x": 0,
          "y": 8
        }
      }
    ],
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "refresh": "5s"
  }
}`

	return ioutil.WriteFile(nextjsDashboardFile, []byte(content), 0644)
}

func (lsm *LoggingStackManager) PullDockerImages() error {
	fmt.Println("📦 Pulling Docker images...")
	
	images := []string{
		"grafana/loki:2.9.0",
		"grafana/promtail:2.9.0",
		"grafana/grafana:10.1.0",
	}
	
	for _, image := range images {
		fmt.Printf("Pulling %s...\n", image)
		cmd := exec.Command("docker", "pull", image)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to pull image %s: %v", image, err)
		}
	}
	
	fmt.Println("✅ All Docker images pulled successfully")
	return nil
}

func (lsm *LoggingStackManager) StopExistingContainers() error {
	fmt.Println("🛑 Stopping existing containers...")
	
	containers := []string{"loki", "promtail", "grafana"}
	
	for _, container := range containers {
		cmd := exec.Command("docker", "stop", container)
		cmd.Run() // Ignore errors if container doesn't exist
		
		cmd = exec.Command("docker", "rm", container)
		cmd.Run() // Ignore errors if container doesn't exist
	}
	
	fmt.Println("✅ Existing containers stopped and removed")
	return nil
}

func (lsm *LoggingStackManager) StartLoggingStack() error {
	fmt.Println("🚀 Starting logging stack...")
	
	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start logging stack: %v", err)
	}
	
	fmt.Println("✅ Logging stack started successfully")
	return nil
}

func (lsm *LoggingStackManager) WaitForServices() error {
	fmt.Println("⏳ Waiting for services to be ready...")
	
	// Wait for Loki to be ready
	for i := 0; i < 30; i++ {
		cmd := exec.Command("curl", "-f", "http://localhost:3100/ready")
		if err := cmd.Run(); err == nil {
			fmt.Println("✅ Loki is ready")
			break
		}
		time.Sleep(2 * time.Second)
	}
	
	// Wait for Grafana to be ready
	for i := 0; i < 30; i++ {
		cmd := exec.Command("curl", "-f", "http://localhost:3001/api/health")
		if err := cmd.Run(); err == nil {
			fmt.Println("✅ Grafana is ready")
			break
		}
		time.Sleep(2 * time.Second)
	}
	
	return nil
}

func (lsm *LoggingStackManager) ShowStatus() error {
	fmt.Println("\n📊 Container Status:")
	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "ps")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to show container status: %v", err)
	}
	
	return nil
}

func (lsm *LoggingStackManager) CreateTestLog() error {
	fmt.Println("📝 Creating test log entry...")
	
	testLog := fmt.Sprintf(`{"level":"info","message":"Test log entry from Go manager","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	
	// Append to log file
	logFile := "/logs/app.log"
	cmd := exec.Command("sudo", "bash", "-c", fmt.Sprintf("echo '%s' >> %s", testLog, logFile))
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create test log: %v", err)
	}
	
	fmt.Println("✅ Test log created")
	return nil
}

func (lsm *LoggingStackManager) PrintAccessInfo() {
	fmt.Println("\n🎉 Logging stack is ready!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 Grafana Dashboard: http://localhost:3001\n")
	fmt.Printf("👤 Username: admin\n")
	fmt.Printf("🔐 Password: admin123\n")
	fmt.Printf("🔍 Loki API: http://localhost:3100\n")
	fmt.Printf("📁 Log File: /logs/app.log\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\n💡 Commands:")
	fmt.Println("  View logs: docker-compose -f docker-compose.logging.yml logs -f")
	fmt.Println("  Stop stack: docker-compose -f docker-compose.logging.yml down")
	fmt.Println("  Restart stack: docker-compose -f docker-compose.logging.yml restart")
}

func main() {
	fmt.Println("🚀 Starting Logging Stack Manager")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	lsm := NewLoggingStackManager()
	
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Creating directories", lsm.CreateDirectories},
		{"Creating Docker Compose file", lsm.CreateDockerComposeFile},
		{"Creating Loki configuration", lsm.CreateLokiConfig},
		{"Creating Promtail configuration", lsm.CreatePromtailConfig},
		{"Creating Grafana datasource", lsm.CreateGrafanaDataSource},
		{"Creating Grafana dashboard config", lsm.CreateGrafanaDashboardConfig},
		{"Creating Next.js dashboard", lsm.CreateNextJSDashboard},
		{"Pulling Docker images", lsm.PullDockerImages},
		{"Stopping existing containers", lsm.StopExistingContainers},
		{"Starting logging stack", lsm.StartLoggingStack},
		{"Waiting for services", lsm.WaitForServices},
		{"Creating test log", lsm.CreateTestLog},
		{"Showing container status", lsm.ShowStatus},
	}
	
	for _, step := range steps {
		if err := step.fn(); err != nil {
			log.Fatalf("❌ Failed at step '%s': %v", step.name, err)
		}
	}
	
	lsm.PrintAccessInfo()
}