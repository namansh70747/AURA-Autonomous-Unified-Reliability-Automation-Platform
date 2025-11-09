package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/analyzer"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/core"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/observer"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// Get config path from environment variable, default to configs/aura.yaml
	configPath := os.Getenv("AURA_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/aura.yaml"
	}

	config, err := core.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Config load failed: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Initialize(config.App.LogLevel); err != nil {
		fmt.Printf("Logger init failed: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	db, err := storage.NewPostgresClient(config.GetDatabaseURL(), logger.Log)
	if err != nil {
		logger.Fatal("Database connection failed", zap.Error(err))
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Health(ctx); err != nil {
		logger.Fatal("Database health check failed", zap.Error(err))
	}

	k8sNamespace := config.Kubernetes.Namespace
	if k8sNamespace == "" {
		k8sNamespace = "default"
	}

	metricsObserver, err := observer.NewMetricsObserver(
		config.Prometheus.URL,
		10*time.Second,
		k8sNamespace,
		db,
		logger.Log,
	) //metriObserver start kardiya here
	if err != nil {
		logger.Fatal("Metrics observer init failed", zap.Error(err))
	}

	// Initialize Pattern Analyzer (Phase 2)
	patternAnalyzer := analyzer.NewAnalyzer(db)
	logger.Info("Pattern analyzer initialized successfully")

	observerCtx, observerCancel := context.WithCancel(context.Background())
	defer observerCancel()

	// Start metrics observer which internally starts both Prometheus and Kubernetes watchers
	go func() {
		if err := metricsObserver.Start(observerCtx); err != nil && err != context.Canceled {
			logger.Error("Observer error", zap.Error(err))
		}
	}()

	// Log Kubernetes watcher status
	if config.Kubernetes.Enabled {
		logger.Info("Kubernetes watcher initialized and started", zap.String("namespace", k8sNamespace))
	} else {
		logger.Info("Kubernetes watcher disabled in config")
	}

	go startConsoleMonitor(db, logger.Log)

	if config.App.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery(), ginLogger())

	router.GET("/health", healthHandler(db, config))
	router.GET("/ready", readyHandler(db))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1")
	{
		v1.GET("/status", statusHandler(config))

		// Metrics endpoints
		v1.GET("/metrics/:service", getServiceMetricsHandler(db))
		v1.GET("/metrics/:service/:metric/stats", getMetricStatsHandler(db))
		v1.GET("/metrics/:service/history", getMetricHistoryHandler(db))
		v1.GET("/metrics/services", getAllServicesHandler(db))

		// Decision endpoints
		v1.GET("/decisions", getDecisionsHandler(db))
		v1.GET("/decisions/stats", getDecisionStatsHandler(db))
		v1.GET("/decisions/:id", getDecisionByIdHandler(db))

		// Observer endpoints
		v1.GET("/observer/health", observerHealthHandler())
		v1.GET("/observer/metrics", observerMetricsHandler(metricsObserver))

		// Kubernetes endpoints
		v1.GET("/kubernetes/pods", getPodsHandler(metricsObserver))
		v1.GET("/kubernetes/pods/:name", getPodDetailHandler(metricsObserver))
		v1.GET("/kubernetes/pods/:name/metrics", getPodMetricsHandler(db))
		v1.GET("/kubernetes/events", getEventsHandler(db))
		v1.GET("/kubernetes/events/:podname", getPodEventsHandler(db))
		v1.GET("/kubernetes/namespace/summary", getNamespaceSummaryHandler(metricsObserver, db))

		// Prometheus endpoints
		v1.GET("/prometheus/health", prometheusHealthHandler(metricsObserver))
		v1.GET("/prometheus/targets", prometheusTargetsHandler(metricsObserver))
		v1.GET("/prometheus/query", prometheusQueryHandler(metricsObserver))
		v1.GET("/prometheus/metrics/summary", prometheusMetricsSummaryHandler(db))

		// Phase 2: Pattern Analysis Endpoints
		v1.GET("/analyze/:service", analyzeServiceHandler(patternAnalyzer))
		v1.GET("/analyze/all", analyzeAllServicesHandler(patternAnalyzer, db))
		v1.GET("/diagnoses/:service", getDiagnosisHistoryHandler(db))
		v1.GET("/diagnoses", getAllDiagnosesHandler(db))

		// Phase 2: Core Detection Endpoints
		v1.GET("/detect/memory-leak/:service", detectMemoryLeakHandler(patternAnalyzer))
		v1.GET("/detect/deployment-bug/:service", detectDeploymentBugHandler(patternAnalyzer))
		v1.GET("/detect/cascade/:service", detectCascadeHandler(patternAnalyzer))
		v1.GET("/detect/resource-exhaustion/:service", detectResourceExhaustionHandler(patternAnalyzer))
		v1.GET("/detect/external-failure/:service", detectExternalFailureHandler(patternAnalyzer))

		// Phase 3: Advanced Analyzer Endpoints
		v1.GET("/advanced/diagnose/:service", analyzeServiceAdvancedHandler(patternAnalyzer))
		v1.GET("/advanced/health/:service", getHealthScoreHandler(patternAnalyzer))
		v1.GET("/advanced/compare", compareServicesHandler(patternAnalyzer))
	}

	srv := &http.Server{
		Addr:           ":8081",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		logger.Info("HTTP server started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	if config.Decision.DryRun {
		logger.Warn("DRY-RUN MODE")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	srv.Shutdown(shutdownCtx)
	observerCancel()
	db.Close()
}

func startConsoleMonitor(db *storage.PostgresClient, log *zap.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		cpuMetric, _ := db.GetLatestMetric(ctx, "sample-app", "cpu_usage")
		memMetric, _ := db.GetLatestMetric(ctx, "sample-app", "memory_usage")

		if cpuMetric != nil && memMetric != nil {
			fmt.Printf("[%s] CPU: %.2f%% | Mem: %.2f%%\n",
				time.Now().Format("15:04:05"), cpuMetric.MetricValue, memMetric.MetricValue)

			log.Info("Metrics",
				zap.String("service", "sample-app"),
				zap.Float64("cpu", cpuMetric.MetricValue),
				zap.Float64("mem", memMetric.MetricValue),
			)
		}

		cancel()
	}
}

func healthHandler(db *storage.PostgresClient, config *core.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := db.Health(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   config.App.Version,
		})
	}
}

func readyHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := db.Health(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"reason": "database unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func statusHandler(config *core.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   config.App.Name,
			"version":   config.App.Version,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func getServiceMetricsHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// Try both metric name formats (with and without _total suffix)
		metricVariants := map[string][]string{
			"cpu_usage":     {"cpu_usage", "cpu_usage_percent"},
			"memory_usage":  {"memory_usage", "memory_usage_percent"},
			"http_requests": {"http_requests", "http_requests_total"},
		}

		currentMetrics := make(map[string]float64)
		failedMetrics := []string{}

		for displayName, variants := range metricVariants {
			metricFound := false

			for _, metricType := range variants {
				metric, err := db.GetLatestMetric(ctx, serviceName, metricType)
				if err == nil && metric != nil {
					currentMetrics[displayName] = metric.MetricValue
					metricFound = true
					logger.Debug("Metric found",
						zap.String("service", serviceName),
						zap.String("display_name", displayName),
						zap.String("actual_name", metricType),
						zap.Float64("value", metric.MetricValue),
					)
					break
				}
			}

			if !metricFound {
				failedMetrics = append(failedMetrics, displayName)
				logger.Debug("No metric data found",
					zap.String("service", serviceName),
					zap.String("metric", displayName),
					zap.Strings("tried_variants", variants),
				)
			}
		}

		if len(currentMetrics) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "no metrics found for service",
			})
			return
		}

		response := gin.H{
			"service_name": serviceName,
			"timestamp":    time.Now().Format(time.RFC3339),
			"metrics":      currentMetrics,
		}

		// Include information about which metrics were missing
		if len(failedMetrics) > 0 {
			response["missing_metrics"] = failedMetrics
		}

		c.JSON(http.StatusOK, response)
	}
}

func getMetricStatsHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		metricName := c.Param("metric")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		stats, err := db.GetMetricStatistics(ctx, serviceName, metricName, 1*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

func getDecisionsHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		decisions, err := db.GetRecentDecisions(ctx, 20)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"decisions": decisions,
			"count":     len(decisions),
		})
	}
}

func getDecisionStatsHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		stats, err := db.GetDecisionStats(ctx, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

func observerHealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "running",
			"interval": "10s",
		})
	}
}

func getPodsHandler(observer *observer.MetricsObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		pods, err := observer.GetKubernetesPods(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Kubernetes not available: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pods":  pods,
			"count": len(pods),
		})
	}
}

func getEventsHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		events, err := db.GetRecentEvents(ctx, "default", 1*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"count":  len(events),
		})
	}
}

func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)

		logger.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("ip", c.ClientIP()),
		)
	}
}

// Enhanced Metrics Handlers

func getMetricHistoryHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		metricType := c.DefaultQuery("type", "cpu_usage")
		durationStr := c.DefaultQuery("duration", "1h")

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid duration format. Use format like: 1h, 30m, 24h",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		metrics, err := db.GetRecentMetrics(ctx, serviceName, metricType, duration)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve metric history",
			})
			return
		}

		if len(metrics) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No metrics found for the specified parameters",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"service":     serviceName,
			"metric_type": metricType,
			"duration":    durationStr,
			"data_points": len(metrics),
			"metrics":     metrics,
			"timestamp":   time.Now().Format(time.RFC3339),
		})
	}
}

func getAllServicesHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		services, err := db.GetAllServices(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve services",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"services":  services,
			"count":     len(services),
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func getDecisionByIdHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		decision, err := db.GetDecisionById(ctx, idStr)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("Decision with ID %s not found", idStr),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"decision":  decision,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func observerMetricsHandler(observer *observer.MetricsObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.DefaultQuery("service", "sample-app")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		metrics, err := observer.GetCurrentMetrics(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve observer metrics",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"service":   serviceName,
			"metrics":   metrics,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// Kubernetes Handlers

func getPodDetailHandler(observer *observer.MetricsObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		podName := c.Param("name")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		pods, err := observer.GetKubernetesPods(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Kubernetes not available or connection failed",
			})
			return
		}

		for _, pod := range pods {
			if pod.Name == podName {
				c.JSON(http.StatusOK, gin.H{
					"pod":       pod,
					"timestamp": time.Now().Format(time.RFC3339),
				})
				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Pod %s not found", podName),
		})
	}
}

func getPodMetricsHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		podName := c.Param("name")
		durationStr := c.DefaultQuery("duration", "1h")

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid duration format",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		metricTypes := []string{"pod_status", "pod_restarts"}
		podMetrics := make(map[string]interface{})

		for _, metricType := range metricTypes {
			metrics, err := db.GetRecentMetrics(ctx, podName, metricType, duration)
			if err != nil {
				continue
			}
			podMetrics[metricType] = metrics
		}

		if len(podMetrics) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("No metrics found for pod %s", podName),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pod":       podName,
			"duration":  durationStr,
			"metrics":   podMetrics,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func getPodEventsHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		podName := c.Param("podname")
		durationStr := c.DefaultQuery("duration", "1h")

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid duration format",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		events, err := db.GetPodEvents(ctx, podName, duration)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve pod events",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pod":       podName,
			"duration":  durationStr,
			"events":    events,
			"count":     len(events),
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func getNamespaceSummaryHandler(observer *observer.MetricsObserver, db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.DefaultQuery("namespace", "default")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		pods, err := observer.GetKubernetesPods(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Kubernetes not available",
			})
			return
		}

		summary := map[string]interface{}{
			"namespace":      namespace,
			"total_pods":     len(pods),
			"running_pods":   0,
			"pending_pods":   0,
			"failed_pods":    0,
			"total_restarts": 0,
		}

		for _, pod := range pods {
			switch pod.Phase {
			case "Running":
				summary["running_pods"] = summary["running_pods"].(int) + 1
			case "Pending":
				summary["pending_pods"] = summary["pending_pods"].(int) + 1
			case "Failed":
				summary["failed_pods"] = summary["failed_pods"].(int) + 1
			}
			summary["total_restarts"] = summary["total_restarts"].(int) + int(pod.Restarts)
		}

		events, _ := db.GetRecentEvents(ctx, namespace, 1*time.Hour)
		summary["recent_events"] = len(events)

		c.JSON(http.StatusOK, gin.H{
			"summary":   summary,
			"pods":      pods,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// Prometheus Handlers

func prometheusHealthHandler(observer *observer.MetricsObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		err := observer.Health(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now().Format(time.RFC3339),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"message":   "Prometheus is reachable",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func prometheusTargetsHandler(observer *observer.MetricsObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		err := observer.Health(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Prometheus not available",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"targets": []map[string]interface{}{
				{
					"name":   "sample-app",
					"url":    "http://sample-app:8080/metrics",
					"status": "up",
				},
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func prometheusQueryHandler(observer *observer.MetricsObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("query")
		service := c.DefaultQuery("service", "sample-app")

		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Query parameter is required. Example: ?query=cpu_usage",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		metrics, err := observer.GetCurrentMetrics(ctx, service)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to execute query",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"query":     query,
			"service":   service,
			"result":    metrics,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func prometheusMetricsSummaryHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		durationStr := c.DefaultQuery("duration", "1h")

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid duration format",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		services := []string{"sample-app"}
		metricTypes := []string{"cpu_usage", "memory_usage", "http_requests"}

		summary := make(map[string]map[string]interface{})

		for _, service := range services {
			summary[service] = make(map[string]interface{})
			for _, metricType := range metricTypes {
				stats, err := db.GetMetricStatistics(ctx, service, metricType, duration)
				if err != nil {
					continue
				}
				summary[service][metricType] = stats
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"duration":  durationStr,
			"summary":   summary,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// analyzeServiceHandler triggers analysis for a specific service
func analyzeServiceHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		logger.Info("Analyzing service via API",
			zap.String("service", serviceName),
			zap.String("client_ip", c.ClientIP()),
		)

		diagnosis, err := analyzer.AnalyzeService(ctx, serviceName)
		if err != nil {
			logger.Error("Analysis failed",
				zap.String("service", serviceName),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Analysis failed: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"service":     serviceName,
			"diagnosis":   diagnosis,
			"analyzed_at": time.Now().Format(time.RFC3339),
		})
	}
}

// analyzeAllServicesHandler analyzes all known services
func analyzeAllServicesHandler(analyzer *analyzer.Analyzer, db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		logger.Info("Analyzing all services via API",
			zap.String("client_ip", c.ClientIP()),
		)

		// Get list of services from database
		services, err := db.GetAllServices(ctx)
		if err != nil {
			logger.Error("Failed to get services list", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get services list",
			})
			return
		}

		if len(services) == 0 {
			logger.Warn("No services found in database")
			c.JSON(http.StatusOK, gin.H{
				"total_services": 0,
				"services":       []string{},
				"diagnoses":      map[string]interface{}{},
				"message":        "No services found. Ensure metrics are being collected.",
			})
			return
		}

		// Analyze all services
		results, err := analyzer.AnalyzeAllServices(ctx, services)
		if err != nil {
			logger.Error("Bulk analysis failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Analysis failed",
			})
			return
		}

		logger.Info("Bulk analysis complete",
			zap.Int("services_analyzed", len(results)),
		)

		c.JSON(http.StatusOK, gin.H{
			"total_services": len(services),
			"services":       services,
			"diagnoses":      results,
			"analyzed_at":    time.Now().Format(time.RFC3339),
		})
	}
}

// getDiagnosisHistoryHandler retrieves diagnosis history for a specific service
func getDiagnosisHistoryHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		limit := 10

		if val, ok := c.GetQuery("limit"); ok {
			if l, parseErr := fmt.Sscanf(val, "%d", &limit); parseErr == nil && l == 1 {
				// limit parsed successfully
			}
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		diagnoses, err := db.GetRecentDiagnosis(ctx, serviceName, limit)
		if err != nil {
			logger.Error("Failed to fetch diagnoses",
				zap.String("service", serviceName),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to fetch diagnoses",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"service":   serviceName,
			"count":     len(diagnoses),
			"diagnoses": diagnoses,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// getAllDiagnosesHandler retrieves all recent diagnoses across all services
func getAllDiagnosesHandler(db *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50
		if val, ok := c.GetQuery("limit"); ok {
			if l, parseErr := fmt.Sscanf(val, "%d", &limit); parseErr == nil && l == 1 {
				// limit parsed successfully
			}
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// Get all diagnoses from database - need to implement this
		// For now, get recent diagnoses for known services
		services, err := db.GetAllServices(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to fetch services",
			})
			return
		}

		allDiagnoses := make(map[string][]*storage.DiagnosisRecord)
		totalCount := 0

		for _, service := range services {
			diagnoses, err := db.GetRecentDiagnosis(ctx, service, limit/len(services))
			if err != nil {
				continue
			}
			allDiagnoses[service] = diagnoses
			totalCount += len(diagnoses)
		}

		c.JSON(http.StatusOK, gin.H{
			"total_count": totalCount,
			"services":    allDiagnoses,
			"timestamp":   time.Now().Format(time.RFC3339),
		})
	}
}

// ====================
// Phase 2: Core Detection Handlers
// ====================

// detectMemoryLeakHandler detects memory leaks
func detectMemoryLeakHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		diagnosis, err := analyzer.AnalyzeService(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Find memory leak detection
		for _, d := range diagnosis.AllDetections {
			if d.Type == "MEMORY_LEAK" {
				c.JSON(http.StatusOK, d)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"type":       "memory_leak",
			"service":    serviceName,
			"detected":   false,
			"confidence": 0,
			"message":    "No memory leak detected",
		})
	}
}

// detectDeploymentBugHandler detects deployment bugs
func detectDeploymentBugHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		diagnosis, err := analyzer.AnalyzeService(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, d := range diagnosis.AllDetections {
			if d.Type == "DEPLOYMENT_BUG" {
				c.JSON(http.StatusOK, d)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"type":       "deployment_bug",
			"service":    serviceName,
			"detected":   false,
			"confidence": 0,
			"message":    "No deployment bug detected",
		})
	}
}

// detectCascadeHandler detects cascade failures
func detectCascadeHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		diagnosis, err := analyzer.AnalyzeService(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, d := range diagnosis.AllDetections {
			if d.Type == "CASCADING_FAILURE" {
				c.JSON(http.StatusOK, d)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"type":       "cascade_failure",
			"service":    serviceName,
			"detected":   false,
			"confidence": 0,
			"message":    "No cascade failure detected",
		})
	}
}

// detectResourceExhaustionHandler detects resource exhaustion
func detectResourceExhaustionHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		diagnosis, err := analyzer.AnalyzeService(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, d := range diagnosis.AllDetections {
			if d.Type == "RESOURCE_EXHAUSTION" {
				c.JSON(http.StatusOK, d)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"type":       "resource_exhaustion",
			"service":    serviceName,
			"detected":   false,
			"confidence": 0,
			"message":    "No resource exhaustion detected",
		})
	}
}

// detectExternalFailureHandler detects external failures
func detectExternalFailureHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		diagnosis, err := analyzer.AnalyzeService(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, d := range diagnosis.AllDetections {
			if d.Type == "EXTERNAL_FAILURE" {
				c.JSON(http.StatusOK, d)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"type":       "external_failure",
			"service":    serviceName,
			"detected":   false,
			"confidence": 0,
			"message":    "No external failure detected",
		})
	}
}

// ==================== ADVANCED ANALYZER ENDPOINTS ====================

func analyzeServiceAdvancedHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		advancedDiag, err := analyzer.AnalyzeServiceAdvanced(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, advancedDiag)
	}
}

func getHealthScoreHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		healthScore, err := analyzer.GetHealthScore(ctx, serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		status := "healthy"
		if healthScore < 50 {
			status = "critical"
		} else if healthScore < 70 {
			status = "degraded"
		} else if healthScore < 90 {
			status = "warning"
		}

		c.JSON(http.StatusOK, gin.H{
			"service":      serviceName,
			"health_score": healthScore,
			"status":       status,
			"timestamp":    time.Now().Format(time.RFC3339),
		})
	}
}

func compareServicesHandler(analyzer *analyzer.Analyzer) gin.HandlerFunc {
	return func(c *gin.Context) {
		servicesParam := c.Query("services")
		if servicesParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "services parameter required (comma-separated list)"})
			return
		}

		services := []string{}
		for _, s := range strings.Split(servicesParam, ",") {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				services = append(services, trimmed)
			}
		}

		if len(services) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no valid services provided"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		comparisons, err := analyzer.CompareServices(ctx, services)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total_services": len(comparisons),
			"timestamp":      time.Now().Format(time.RFC3339),
			"comparisons":    comparisons,
		})
	}
}
