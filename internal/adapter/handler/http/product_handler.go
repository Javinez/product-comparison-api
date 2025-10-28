package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	
	"github.com/javinez/product-comparison-api/internal/domain"
	"github.com/javinez/product-comparison-api/internal/usecase"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// ProductHandler handles HTTP requests for product operations
type ProductHandler struct {
	interactor *usecase.ProductInteractor
	logger     *zap.Logger
}

// NewProductHandler creates a new ProductHandler instance
func NewProductHandler(interactor *usecase.ProductInteractor, logger *zap.Logger) *ProductHandler {
	return &ProductHandler{
		interactor: interactor,
		logger:     logger,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
}

// SuccessResponse represents a successful response
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Success   bool        `json:"success"`
}

// RegisterRoutes registers all HTTP routes for the product handler
func (h *ProductHandler) RegisterRoutes(router *mux.Router) {
	router.Use(h.loggingMiddleware)
	router.Use(h.corsMiddleware)
	router.Use(h.rateLimitMiddleware)
	
	// Health check
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	
	// Product endpoints
	router.HandleFunc("/api/v1/products", h.GetAllProducts).Methods("GET")
	router.HandleFunc("/api/v1/products/compare", h.CompareProducts).Methods("GET")
	router.HandleFunc("/api/v1/products/search", h.SearchProducts).Methods("GET")
	router.HandleFunc("/api/v1/products/category/{category}", h.GetProductsByCategory).Methods("GET")
	router.HandleFunc("/api/v1/products/{id}", h.GetProduct).Methods("GET")
	router.HandleFunc("/api/v1/products", h.CreateProduct).Methods("POST")
	router.HandleFunc("/api/v1/products/{id}", h.UpdateProduct).Methods("PUT")
	router.HandleFunc("/api/v1/products/{id}", h.DeleteProduct).Methods("DELETE")
}

// HealthCheck handles health check requests
func (h *ProductHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.sendJSONResponse(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// GetProduct handles GET /api/v1/products/{id}
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if id == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Product ID is required", r.URL.Path)
		return
	}
	
	product, err := h.interactor.GetProduct(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			h.sendErrorResponse(w, http.StatusNotFound, "Product not found", r.URL.Path)
			return
		}
		h.logger.Error("Failed to get product", zap.Error(err))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Internal server error", r.URL.Path)
		return
	}
	
	h.sendSuccessResponse(w, http.StatusOK, product)
}

// CompareProducts handles GET /api/v1/products/compare?ids=id1,id2,id3
func (h *ProductHandler) CompareProducts(w http.ResponseWriter, r *http.Request) {
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Product IDs are required (e.g., ?ids=id1,id2,id3)", r.URL.Path)
		return
	}
	
	// Parse product IDs
	ids := strings.Split(idsParam, ",")
	if len(ids) < 2 {
		h.sendErrorResponse(w, http.StatusBadRequest, "At least 2 product IDs are required for comparison", r.URL.Path)
		return
	}
	
	// Limit the number of products that can be compared
	if len(ids) > 10 {
		h.sendErrorResponse(w, http.StatusBadRequest, "Maximum 10 products can be compared at once", r.URL.Path)
		return
	}
	
	// Clean up IDs
	for i := range ids {
		ids[i] = strings.TrimSpace(ids[i])
	}
	
	comparison, err := h.interactor.CompareProducts(r.Context(), ids)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidComparison) {
			h.sendErrorResponse(w, http.StatusBadRequest, err.Error(), r.URL.Path)
			return
		}
		h.logger.Error("Failed to compare products", zap.Error(err), zap.Strings("ids", ids))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to compare products", r.URL.Path)
		return
	}
	
	h.sendSuccessResponse(w, http.StatusOK, comparison)
}

// GetAllProducts handles GET /api/v1/products
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := 10 // Default limit
	offset := 0 // Default offset
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	
	products, err := h.interactor.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get all products", zap.Error(err))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve products", r.URL.Path)
		return
	}
	
	response := map[string]interface{}{
		"products": products,
		"meta": map[string]int{
			"limit":  limit,
			"offset": offset,
			"count":  len(products),
		},
	}
	
	h.sendSuccessResponse(w, http.StatusOK, response)
}

// GetProductsByCategory handles GET /api/v1/products/category/{category}
func (h *ProductHandler) GetProductsByCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars["category"]
	
	if category == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Category is required", r.URL.Path)
		return
	}
	
	products, err := h.interactor.GetProductsByCategory(r.Context(), category)
	if err != nil {
		h.logger.Error("Failed to get products by category", zap.Error(err), zap.String("category", category))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve products", r.URL.Path)
		return
	}
	
	h.sendSuccessResponse(w, http.StatusOK, products)
}

// SearchProducts handles GET /api/v1/products/search?q=query
func (h *ProductHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Search query is required (e.g., ?q=laptop)", r.URL.Path)
		return
	}
	
	// Minimum query length
	if len(query) < 2 {
		h.sendErrorResponse(w, http.StatusBadRequest, "Search query must be at least 2 characters long", r.URL.Path)
		return
	}
	
	products, err := h.interactor.SearchProducts(r.Context(), query)
	if err != nil {
		h.logger.Error("Failed to search products", zap.Error(err), zap.String("query", query))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to search products", r.URL.Path)
		return
	}
	
	h.sendSuccessResponse(w, http.StatusOK, products)
}

// CreateProduct handles POST /api/v1/products
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var product domain.Product
	
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request body", r.URL.Path)
		return
	}
	
	if err := h.interactor.SaveProduct(r.Context(), &product); err != nil {
		if errors.Is(err, domain.ErrInvalidProductID) || 
		   errors.Is(err, domain.ErrEmptyProductName) ||
		   errors.Is(err, domain.ErrInvalidPrice) ||
		   errors.Is(err, domain.ErrInvalidRating) {
			h.sendErrorResponse(w, http.StatusBadRequest, err.Error(), r.URL.Path)
			return
		}
		h.logger.Error("Failed to create product", zap.Error(err))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to create product", r.URL.Path)
		return
	}
	
	h.sendSuccessResponse(w, http.StatusCreated, product)
}

// UpdateProduct handles PUT /api/v1/products/{id}
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if id == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Product ID is required", r.URL.Path)
		return
	}
	
	var product domain.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request body", r.URL.Path)
		return
	}
	
	product.ID = id
	
	if err := h.interactor.UpdateProduct(r.Context(), &product); err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			h.sendErrorResponse(w, http.StatusNotFound, "Product not found", r.URL.Path)
			return
		}
		h.logger.Error("Failed to update product", zap.Error(err))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to update product", r.URL.Path)
		return
	}
	
	h.sendSuccessResponse(w, http.StatusOK, product)
}

// DeleteProduct handles DELETE /api/v1/products/{id}
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if id == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Product ID is required", r.URL.Path)
		return
	}
	
	if err := h.interactor.DeleteProduct(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			h.sendErrorResponse(w, http.StatusNotFound, "Product not found", r.URL.Path)
			return
		}
		h.logger.Error("Failed to delete product", zap.Error(err))
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to delete product", r.URL.Path)
		return
	}
	
	h.sendSuccessResponse(w, http.StatusOK, map[string]string{"message": "Product deleted successfully"})
}

// Middleware functions

func (h *ProductHandler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		next.ServeHTTP(wrapped, r)
		
		h.logger.Info("HTTP Request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", time.Since(start)),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}

func (h *ProductHandler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (h *ProductHandler) rateLimitMiddleware(next http.Handler) http.Handler {
	// Simple in-memory rate limiter (should use Redis in production)
	// This is a placeholder implementation
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement proper rate limiting with Redis or similar
		next.ServeHTTP(w, r)
	})
}

// Helper functions

func (h *ProductHandler) sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

func (h *ProductHandler) sendSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	response := SuccessResponse{
		Data:      data,
		Timestamp: time.Now(),
		Success:   true,
	}
	h.sendJSONResponse(w, statusCode, response)
}

func (h *ProductHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string, path string) {
	response := ErrorResponse{
		Error:     http.StatusText(statusCode),
		Message:   message,
		Timestamp: time.Now(),
		Path:      path,
	}
	h.sendJSONResponse(w, statusCode, response)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
