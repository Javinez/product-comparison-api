package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	
	"github.com/javinez/product-comparison-api/pkg/config"
	httpHandler "github.com/javinez/product-comparison-api/internal/adapter/handler/http"
	jsonRepo "github.com/javinez/product-comparison-api/internal/adapter/repository/json"
	"github.com/javinez/product-comparison-api/internal/usecase"
)

func main() {
	// Load configuration
	cfg := config.Load()
	
	// Initialize logger
	logger, err := initLogger(cfg.Logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()
	
	logger.Info("Starting Product Comparison API",
		zap.String("version", "1.0.0"),
		zap.String("environment", getEnv("ENVIRONMENT", "development")),
	)
	
	// Initialize repository
	repository, err := initRepository(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to initialize repository", zap.Error(err))
	}
	
	// Initialize cache service (optional)
	var cacheService usecase.CacheService
	if cfg.Cache.Enabled {
		cacheService = initCacheService(cfg.Cache, logger)
	}
	
	// Initialize use case interactor
	productInteractor := usecase.NewProductInteractor(repository, cacheService)
	
	// Initialize HTTP handler
	productHandler := httpHandler.NewProductHandler(productInteractor, logger)
	
	// Setup routes
	router := mux.NewRouter()
	productHandler.RegisterRoutes(router)
	
	// Add prometheus metrics endpoint
	router.HandleFunc("/metrics", metricsHandler).Methods("GET")
	
	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	
	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server starting",
			zap.String("address", srv.Addr),
			zap.Bool("cors_enabled", cfg.Server.EnableCORS),
		)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()
	
	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Info("Shutting down server...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}
	
	logger.Info("Server shutdown complete")
}

func initLogger(cfg config.LoggerConfig) (*zap.Logger, error) {
	// Parse log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	
	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	// Create encoder based on format
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	
	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
	
	// Create logger with options
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	
	return logger, nil
}

func initRepository(cfg config.DatabaseConfig, logger *zap.Logger) (usecase.ProductRepository, error) {
	switch cfg.Type {
	case "json":
		logger.Info("Using JSON file repository", zap.String("path", cfg.DataPath))
		return jsonRepo.NewJSONProductRepository(cfg.DataPath)
	case "postgres":
		// TODO: Implement PostgreSQL repository
		return nil, fmt.Errorf("PostgreSQL repository not yet implemented")
	case "mongodb":
		// TODO: Implement MongoDB repository
		return nil, fmt.Errorf("MongoDB repository not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

func initCacheService(cfg config.CacheConfig, logger *zap.Logger) usecase.CacheService {
	// For now, return nil (no cache)
	// TODO: Implement Redis cache service
	logger.Warn("Cache service not implemented, running without cache")
	return nil
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement Prometheus metrics
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# Metrics endpoint not yet implemented\n"))
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
