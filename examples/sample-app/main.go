package main
//Working Perfectly 
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
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
		// only goes up
		// .Inc() .Add(x)
	) 
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,//Buckets are now default 
		},
		[]string{"method", "endpoint"},
	)// here we are using observe methid to add values in the prometheus 
	// record measurement value into buckets
	// .Observe(x)
	cpuUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "Simulated CPU usage percentage",
		},// can go up or down, yes we can go up or down 
		// Set(x) .Inc() .Dec() .Add(x) .Sub(x)
	)

	memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_percent",
			Help: "Simulated memory usage percentage",
		},// can go up or down, yes we can go up or down 
		// Set(x) .Inc() .Dec() .Add(x) .Sub(x)
	)

	errorRate = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "app_errors_total",
			Help: "Total number of application errors",
		},
		// only goes up
		// .Inc() .Add(x)
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
	/*
	| mode              | name        | meaning                      |
    | ----------------- | ----------- | ---------------------------- |
    | `gin.DebugMode`   | Development | prints many logs, debug info |
    | `gin.TestMode`    | Testing     | used in unit tests           |
    | `gin.ReleaseMode` | Production  | very few logs, faster        |
	*/
	router := gin.New() //new router is created 

	router.Use(gin.Recovery())
	/*
	without Recovery
	----------------
	panic in handler → whole server stops → crash
	with Recovery
	-------------
	panic in handler → Recovery catches → return 500 → server keeps running

	*/
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
	}// Gracefull Shutdown 
}

func prometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()// Move to the Next Handler 
			return
		}

		start := time.Now() // time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start).Seconds()// time from where it starts in seconds.
		status := c.Writer.Status()// Get the HTTP response status code that the endpoint returned.

		requestCounter.WithLabelValues(c.Request.Method, path, string(rune(status))).Inc()
		requestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
/* 
| code | meaning | example value | why we use it |
|------|---------|---------------|---------------|
| c.Request.Method | HTTP method used for the request | "GET", "POST", "PUT", "DELETE" | to know what type of request came |
| c.Request.URL.Path | only the URL path (no query params) | "/health", "/ready", "/error" | used to track which endpoint was hit |
| c.Request.URL.RawQuery | full query string after "?" | "id=5&name=ram" | to read extra info client sent |
| c.Request.Host | hostname of request | "localhost:8080" | helpful for multi-host setups |
| c.Request.Header | complete header map | map[string][]string{"Content-Type":["application/json"]} | to see all headers like auth token |
| c.Request.Body | raw request body | {"name":"ram","age":21} | used to read JSON/form data sent by client |
*/
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
		cpu := 30.0 + rand.Float64()*40.0 //generate a random CPU usage between 30% to 70%
		cpuUsage.Set(cpu)// set cpuUsage metric to that random number 

		mem := 40.0 + rand.Float64()*40.0 //generate random memory usage between 40% to 80%
		memoryUsage.Set(mem)// set memoryUsage metric to that random number

		if rand.Float64() < 0.1 {
			errorRate.Inc() //.Inc()
		}
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
