package main

//Enhanced Sample App with Test Scenarios for AI Analyzer
import (
	"context"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Test scenario control
var (
	scenarioMutex     sync.RWMutex
	currentScenario   = "normal"
	scenarioStartTime time.Time
	memoryLeakRate    = 0.0
	accumulatedMemory = 50.0
)

var (
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

	// Basic endpoints
	router.GET("/", handleRoot)
	router.GET("/health", handleHealth)
	router.GET("/ready", handleReady)
	router.GET("/error", handleError)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Test scenario control endpoints
	router.POST("/scenario/:name", setScenario)
	router.GET("/scenario", getScenario)

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

		requestCounter.WithLabelValues(c.Request.Method, path, strconv.Itoa(status)).Inc()
		requestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

func handleRoot(c *gin.Context) {
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Enhanced Sample App with AI Test Scenarios",
		"current_scenario": getCurrentScenario(),
		"endpoints": gin.H{
			"metrics":          "/metrics",
			"health":           "/health",
			"ready":            "/ready",
			"error":            "/error",
			"scenario_control": "/scenario/:name (POST)",
			"scenario_status":  "/scenario (GET)",
		},
		"available_scenarios": []string{
			"normal", "memory-leak", "cpu-spike", "error-storm",
			"resource-exhaustion", "deployment-bug", "external-failure", "cascade",
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

// Scenario control handlers
func setScenario(c *gin.Context) {
	scenarioName := c.Param("name")

	scenarioMutex.Lock()
	defer scenarioMutex.Unlock()

	validScenarios := map[string]bool{
		"normal":              true,
		"memory-leak":         true,
		"cpu-spike":           true,
		"error-storm":         true,
		"resource-exhaustion": true,
		"deployment-bug":      true,
		"external-failure":    true,
		"cascade":             true,
	}

	if !validScenarios[scenarioName] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scenario",
			"valid_scenarios": []string{
				"normal", "memory-leak", "cpu-spike", "error-storm",
				"resource-exhaustion", "deployment-bug", "external-failure", "cascade",
			},
		})
		return
	}

	currentScenario = scenarioName
	scenarioStartTime = time.Now()

	// Reset accumulated memory for new scenario
	if scenarioName == "memory-leak" {
		accumulatedMemory = 60.0 // Start at 60%
		memoryLeakRate = 0.021   // 0.021% per 5-sec tick = ~0.25%/min
	} else {
		accumulatedMemory = 50.0
		memoryLeakRate = 0.0
	}

	log.Printf("✅ Scenario activated: %s (started at %s)", scenarioName, scenarioStartTime.Format(time.RFC3339))

	c.JSON(http.StatusOK, gin.H{
		"message":    "Scenario activated",
		"scenario":   scenarioName,
		"started_at": scenarioStartTime.Format(time.RFC3339),
	})
}

func getScenario(c *gin.Context) {
	scenarioMutex.RLock()
	defer scenarioMutex.RUnlock()

	var duration string
	if !scenarioStartTime.IsZero() {
		duration = time.Since(scenarioStartTime).String()
	}

	c.JSON(http.StatusOK, gin.H{
		"current_scenario": currentScenario,
		"started_at":       scenarioStartTime.Format(time.RFC3339),
		"duration":         duration,
	})
}

func getCurrentScenario() string {
	scenarioMutex.RLock()
	defer scenarioMutex.RUnlock()
	return currentScenario
}

func simulateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		scenarioMutex.RLock()
		scenario := currentScenario
		scenarioMutex.RUnlock()

		var cpu, mem float64

		switch scenario {
		case "normal":
			// ✅ FALSE: No detection should trigger
			// Memory stable, CPU normal, minimal errors
			cpu = 35.0 + rand.Float64()*15.0 // 35-50%
			mem = 45.0 + rand.Float64()*15.0 // 45-60%
			// Very rare errors (< 1/min)
			if rand.Float64() < 0.02 {
				errorRate.Inc()
			}

		case "memory-leak":
			// ✅ DETECT: Sustained memory growth >0.15%/min, low volatility, no CPU correlation
			// Detector needs: trend >0.15%/min, volatility <0.15, autocorr >0.8, 2+ quality signals
			elapsedTicks := time.Since(scenarioStartTime).Seconds() / 5.0
			// Leak rate: 0.25%/min = 0.02% per 5-sec tick (above 0.15%/min threshold)
			accumulatedMemory = 60.0 + (elapsedTicks * 0.021) + rand.Float64()*0.5 // Consistent growth
			if accumulatedMemory > 95.0 {
				accumulatedMemory = 95.0
			}
			cpu = 40.0 + rand.Float64()*10.0 // 40-50% - NO correlation with memory
			mem = accumulatedMemory
			// Minimal errors
			if rand.Float64() < 0.03 {
				errorRate.Inc()
			}

		case "cpu-spike":
			// ✅ FALSE: Only CPU high, but memory normal
			// Detector needs BOTH CPU >80% AND Memory >85% for resource exhaustion
			cpu = 88.0 + rand.Float64()*8.0  // 88-96% (high)
			mem = 55.0 + rand.Float64()*15.0 // 55-70% (normal) - NOT BOTH HIGH
			// Some stress errors but not enough for other detections
			if rand.Float64() < 0.25 {
				for i := 0; i < rand.Intn(3)+1; i++ {
					errorRate.Inc()
				}
			}

		case "error-storm":
			// ✅ FALSE: Just errors, but not enough signals for deployment bug
			// Deployment bug needs: spikiness >2.0 AND error rate >5/min AND 2+ quality signals
			// This creates errors but resources are high (fails cross-validation)
			cpu = 75.0 + rand.Float64()*15.0 // 75-90% - HIGH (fails normal resources check)
			mem = 70.0 + rand.Float64()*15.0 // 70-85% - HIGH
			// Generate errors but not extreme spikiness
			errorCount := rand.Intn(8) + 4 // 4-12 errors per 5sec = ~50-144/min
			for i := 0; i < errorCount; i++ {
				errorRate.Inc()
			}

		case "resource-exhaustion":
			// ✅ DETECT: BOTH CPU >80% AND Memory >85%
			// Detector needs: CPU >80%, Memory >85%, both high bonus, 2+ quality signals
			cpu = 87.0 + rand.Float64()*10.0 // 87-97% (above 80%)
			mem = 89.0 + rand.Float64()*8.0  // 89-97% (above 85%)
			// Resource starvation errors
			if rand.Float64() < 0.5 {
				for i := 0; i < rand.Intn(6)+3; i++ {
					errorRate.Inc()
				}
			}

		case "deployment-bug":
			// ✅ DETECT: High errors + spikiness, normal resources, independent of CPU
			// Detector needs: spikiness >2.0, error rate >15/min, CPU <70%, Memory <70%, 2+ quality signals
			cpu = 45.0 + rand.Float64()*15.0 // 45-60% (below 70% - normal resources)
			mem = 50.0 + rand.Float64()*15.0 // 50-65% (below 70% - normal resources)
			// MASSIVE error bursts for high spikiness
			// Generate 20-40 errors per 5sec = 240-480 errors/min (very high rate)
			if rand.Float64() < 0.9 { // 90% of ticks have bursts
				errorCount := rand.Intn(21) + 20 // 20-40 errors
				for i := 0; i < errorCount; i++ {
					errorRate.Inc()
				}
			}

		case "external-failure":
			// ✅ DETECT: High errors + LOW CPU/Memory (external dependency issue)
			// Detector needs: errors >10/min, CPU <65%, latency-error corr >0.6, 3+ quality signals
			cpu = 35.0 + rand.Float64()*20.0 // 35-55% (well below 65%)
			mem = 45.0 + rand.Float64()*20.0 // 45-65% (normal)
			// Consistent moderate-high errors simulating external timeout/failures
			// 12-20 errors per 5sec = 144-240/min
			if rand.Float64() < 0.8 {
				errorCount := rand.Intn(9) + 12 // 12-20 errors
				for i := 0; i < errorCount; i++ {
					errorRate.Inc()
				}
			}

		case "cascade":
			// ✅ DETECT: 3+ degraded metrics (CPU, Memory, Errors, Latency)
			// Detector needs: 3+ degraded (CPU >85%, Memory >88%, Errors >15/min, Latency >2s), 2+ quality signals
			elapsedMinutes := time.Since(scenarioStartTime).Minutes()
			// Gradual degradation reaching critical levels
			degradationFactor := math.Min(elapsedMinutes*15, 50.0)

			cpu = 60.0 + degradationFactor + rand.Float64()*10.0 // Grows to 90-110%
			mem = 60.0 + degradationFactor + rand.Float64()*10.0 // Grows to 90-110%

			if cpu > 98.0 {
				cpu = 98.0
			}
			if mem > 98.0 {
				mem = 98.0
			}

			// Increasing errors as system degrades
			errorProbability := math.Min(0.6+elapsedMinutes*0.2, 0.95)
			if rand.Float64() < errorProbability {
				errorCount := rand.Intn(12) + int(elapsedMinutes*3) + 8 // Grows to 20-30 errors/5sec
				for i := 0; i < errorCount; i++ {
					errorRate.Inc()
				}
			}

		default:
			cpu = 30.0 + rand.Float64()*20.0
			mem = 40.0 + rand.Float64()*20.0
		}

		cpuUsage.Set(cpu)
		memoryUsage.Set(mem)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
