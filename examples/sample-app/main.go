package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Prometheus metrics
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	cpuUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "Simulated CPU usage percentage",
		},
	)

	memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_percent",
			Help: "Simulated memory usage percentage",
		},
	)

	errorRate = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "app_errors_total",
			Help: "Total number of application errors",
		},
	)
)

func init() {
	prometheus.MustRegister(requestCounter)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memoryUsage)
	prometheus.MustRegister(errorRate)
}

func main() {
	go simulateMetrics()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(prometheusMiddleware())

	router.GET("/", handleRoot)
	router.GET("/health", handleHealth)
	router.GET("/ready", handleReady)
	router.GET("/error", handleError)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	port := getEnv("APP_PORT", "8080")

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Printf("Sample App on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

func prometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()

		requestCounter.WithLabelValues(c.Request.Method, path, string(rune(status))).Inc()
		requestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

func handleRoot(c *gin.Context) {
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	c.JSON(http.StatusOK, gin.H{
		"message": "Sample App is running",
		"endpoints": gin.H{
			"metrics": "/metrics",
			"health":  "/health",
			"ready":   "/ready",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func handleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func handleError(c *gin.Context) {
	errorRate.Inc()
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":     "Simulated error",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func simulateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cpu := 30.0 + rand.Float64()*40.0
		cpuUsage.Set(cpu)

		mem := 40.0 + rand.Float64()*40.0
		memoryUsage.Set(mem)

		if rand.Float64() < 0.1 {
			errorRate.Inc()
		}
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
