package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"
	
	"github.com/javinez/product-comparison-api/internal/domain"
	"github.com/google/uuid"
)

// ProductInteractor contains the business logic for product operations
type ProductInteractor struct {
	repository ProductRepository
	cache      CacheService // Optional cache service interface
}

// CacheService defines the interface for caching operations
type CacheService interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

// NewProductInteractor creates a new instance of ProductInteractor
func NewProductInteractor(repo ProductRepository, cache CacheService) *ProductInteractor {
	return &ProductInteractor{
		repository: repo,
		cache:      cache,
	}
}

// GetProduct retrieves a single product by ID
func (pi *ProductInteractor) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	if id == "" {
		return nil, domain.ErrInvalidProductID
	}
	
	// Try to get from cache first if available
	if pi.cache != nil {
		cacheKey := fmt.Sprintf("product:%s", id)
		if cached, err := pi.cache.Get(ctx, cacheKey); err == nil {
			if product, ok := cached.(*domain.Product); ok {
				return product, nil
			}
		}
	}
	
	product, err := pi.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Cache the result if cache is available
	if pi.cache != nil && product != nil {
		cacheKey := fmt.Sprintf("product:%s", id)
		_ = pi.cache.Set(ctx, cacheKey, product, 5*time.Minute)
	}
	
	return product, nil
}

// CompareProducts compares multiple products and returns a comparison result
func (pi *ProductInteractor) CompareProducts(ctx context.Context, productIDs []string) (*domain.ProductComparison, error) {
	if len(productIDs) < 2 {
		return nil, domain.ErrInvalidComparison
	}
	
	// Remove duplicates
	uniqueIDs := removeDuplicates(productIDs)
	
	// Fetch all products
	products, err := pi.repository.FindByIDs(ctx, uniqueIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	
	if len(products) < 2 {
		return nil, domain.ErrInvalidComparison
	}
	
	// Create comparison
	comparison := &domain.ProductComparison{
		Products:     products,
		ComparisonID: uuid.New().String(),
		CreatedAt:    time.Now(),
		Differences:  make(map[string][]string),
		Similarities: make(map[string]interface{}),
	}
	
	// Analyze differences and similarities
	pi.analyzeComparison(comparison)
	
	return comparison, nil
}

// GetProductsByCategory retrieves all products in a specific category
func (pi *ProductInteractor) GetProductsByCategory(ctx context.Context, category string) ([]domain.Product, error) {
	if category == "" {
		return nil, fmt.Errorf("category cannot be empty")
	}
	
	products, err := pi.repository.FindByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products by category: %w", err)
	}
	
	// Sort products by rating (highest first)
	sort.Slice(products, func(i, j int) bool {
		return products[i].Rating > products[j].Rating
	})
	
	return products, nil
}

// SearchProducts searches for products matching the query
func (pi *ProductInteractor) SearchProducts(ctx context.Context, query string) ([]domain.Product, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	
	products, err := pi.repository.SearchProducts(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search products: %w", err)
	}
	
	return products, nil
}

// GetAllProducts retrieves all products with pagination
func (pi *ProductInteractor) GetAllProducts(ctx context.Context, limit, offset int) ([]domain.Product, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}
	
	products, err := pi.repository.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all products: %w", err)
	}
	
	return products, nil
}

// SaveProduct saves a new product
func (pi *ProductInteractor) SaveProduct(ctx context.Context, product *domain.Product) error {
	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}
	
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	
	if product.ID == "" {
		product.ID = uuid.New().String()
	}
	
	err := pi.repository.Save(ctx, product)
	if err != nil {
		return fmt.Errorf("failed to save product: %w", err)
	}
	
	// Invalidate cache if available
	if pi.cache != nil {
		cacheKey := fmt.Sprintf("product:%s", product.ID)
		_ = pi.cache.Delete(ctx, cacheKey)
	}
	
	return nil
}

// UpdateProduct updates an existing product
func (pi *ProductInteractor) UpdateProduct(ctx context.Context, product *domain.Product) error {
	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}
	
	product.UpdatedAt = time.Now()
	
	err := pi.repository.Update(ctx, product)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	
	// Invalidate cache if available
	if pi.cache != nil {
		cacheKey := fmt.Sprintf("product:%s", product.ID)
		_ = pi.cache.Delete(ctx, cacheKey)
	}
	
	return nil
}

// DeleteProduct deletes a product by ID
func (pi *ProductInteractor) DeleteProduct(ctx context.Context, id string) error {
	if id == "" {
		return domain.ErrInvalidProductID
	}
	
	err := pi.repository.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	
	// Invalidate cache if available
	if pi.cache != nil {
		cacheKey := fmt.Sprintf("product:%s", id)
		_ = pi.cache.Delete(ctx, cacheKey)
	}
	
	return nil
}

// analyzeComparison analyzes products to find differences and similarities
func (pi *ProductInteractor) analyzeComparison(comparison *domain.ProductComparison) {
	if len(comparison.Products) < 2 {
		return
	}
	
	// Analyze price range
	var minPrice, maxPrice float64 = comparison.Products[0].Price, comparison.Products[0].Price
	var totalRating float64
	brands := make(map[string]bool)
	categories := make(map[string]bool)
	
	for _, product := range comparison.Products {
		if product.Price < minPrice {
			minPrice = product.Price
		}
		if product.Price > maxPrice {
			maxPrice = product.Price
		}
		totalRating += product.Rating
		brands[product.Brand] = true
		categories[product.Category] = true
	}
	
	// Set similarities and differences
	comparison.Similarities["avg_rating"] = totalRating / float64(len(comparison.Products))
	comparison.Similarities["price_range"] = map[string]float64{
		"min": minPrice,
		"max": maxPrice,
	}
	
	// Collect differences
	if len(brands) > 1 {
		brandList := make([]string, 0, len(brands))
		for brand := range brands {
			brandList = append(brandList, brand)
		}
		comparison.Differences["brands"] = brandList
	}
	
	if len(categories) > 1 {
		categoryList := make([]string, 0, len(categories))
		for category := range categories {
			categoryList = append(categoryList, category)
		}
		comparison.Differences["categories"] = categoryList
	}
	
	// Analyze specifications differences
	specDifferences := pi.analyzeSpecificationDifferences(comparison.Products)
	if len(specDifferences) > 0 {
		comparison.Differences["specifications"] = specDifferences
	}
}

// analyzeSpecificationDifferences compares specifications across products
func (pi *ProductInteractor) analyzeSpecificationDifferences(products []domain.Product) []string {
	if len(products) < 2 {
		return nil
	}
	
	differences := []string{}
	allSpecs := make(map[string][]interface{})
	
	// Collect all specifications
	for _, product := range products {
		for key, value := range product.Specifications {
			allSpecs[key] = append(allSpecs[key], value)
		}
	}
	
	// Find differences
	for key, values := range allSpecs {
		if len(values) != len(products) {
			differences = append(differences, fmt.Sprintf("%s: not all products have this specification", key))
			continue
		}
		
		// Check if all values are the same
		firstValue := fmt.Sprintf("%v", values[0])
		allSame := true
		for i := 1; i < len(values); i++ {
			if fmt.Sprintf("%v", values[i]) != firstValue {
				allSame = false
				break
			}
		}
		
		if !allSame {
			differences = append(differences, fmt.Sprintf("%s: values differ across products", key))
		}
	}
	
	return differences
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(strings []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	
	for _, entry := range strings {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	
	return list
}
