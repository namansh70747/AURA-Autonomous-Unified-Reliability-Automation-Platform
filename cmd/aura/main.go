package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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
	)
	if err != nil {
		logger.Fatal("Metrics observer init failed", zap.Error(err))
	}

	observerCtx, observerCancel := context.WithCancel(context.Background())
	defer observerCancel()

	go func() {
		if err := metricsObserver.Start(observerCtx); err != nil && err != context.Canceled {
			logger.Error("Observer error", zap.Error(err))
		}
	}()

	if config.Kubernetes.Enabled {
		logger.Info("K8s watcher enabled", zap.String("namespace", k8sNamespace))
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

		metricTypes := []string{"cpu_usage", "memory_usage", "http_requests"}
		currentMetrics := make(map[string]float64)

		for _, metricType := range metricTypes {
			metric, err := db.GetLatestMetric(ctx, serviceName, metricType)
			if err != nil {
				continue
			}
			if metric != nil {
				currentMetrics[metricType] = metric.MetricValue
			}
		}

		if len(currentMetrics) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "no metrics found for service",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"service_name": serviceName,
			"timestamp":    time.Now().Format(time.RFC3339),
			"metrics":      currentMetrics,
		})
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
