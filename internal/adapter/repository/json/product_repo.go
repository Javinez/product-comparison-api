package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	
	"github.com/javinez/product-comparison-api/internal/domain"
)

// JSONProductRepository implements ProductRepository using JSON files
type JSONProductRepository struct {
	dataPath string
	mu       sync.RWMutex
	cache    map[string]*domain.Product
}

// NewJSONProductRepository creates a new JSON-based repository
func NewJSONProductRepository(dataPath string) (*JSONProductRepository, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	
	repo := &JSONProductRepository{
		dataPath: dataPath,
		cache:    make(map[string]*domain.Product),
	}
	
	// Load existing products into cache
	if err := repo.loadCache(); err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}
	
	// Initialize with sample data if empty
	if len(repo.cache) == 0 {
		repo.initializeSampleData()
	}
	
	return repo, nil
}

// FindByID retrieves a product by its ID
func (r *JSONProductRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	product, exists := r.cache[id]
	if !exists {
		return nil, domain.ErrProductNotFound
	}
	
	// Return a copy to prevent external modifications
	productCopy := *product
	return &productCopy, nil
}

// FindByIDs retrieves multiple products by their IDs
func (r *JSONProductRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	products := make([]domain.Product, 0, len(ids))
	notFound := []string{}
	
	for _, id := range ids {
		if product, exists := r.cache[id]; exists {
			products = append(products, *product)
		} else {
			notFound = append(notFound, id)
		}
	}
	
	if len(notFound) > 0 {
		return products, fmt.Errorf("products not found: %v", notFound)
	}
	
	return products, nil
}

// FindAll retrieves all products with pagination
func (r *JSONProductRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	products := make([]domain.Product, 0, len(r.cache))
	for _, product := range r.cache {
		products = append(products, *product)
	}
	
	// Apply pagination
	start := offset
	if start > len(products) {
		return []domain.Product{}, nil
	}
	
	end := start + limit
	if end > len(products) {
		end = len(products)
	}
	
	return products[start:end], nil
}

// Save persists a new product
func (r *JSONProductRepository) Save(ctx context.Context, product *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if product already exists
	if _, exists := r.cache[product.ID]; exists {
		return errors.New("product already exists")
	}
	
	// Add to cache
	r.cache[product.ID] = product
	
	// Save to file
	if err := r.saveToFile(product); err != nil {
		delete(r.cache, product.ID) // Rollback cache change
		return fmt.Errorf("failed to save product to file: %w", err)
	}
	
	return nil
}

// Update updates an existing product
func (r *JSONProductRepository) Update(ctx context.Context, product *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if product exists
	if _, exists := r.cache[product.ID]; !exists {
		return domain.ErrProductNotFound
	}
	
	// Update cache
	r.cache[product.ID] = product
	
	// Save to file
	if err := r.saveToFile(product); err != nil {
		return fmt.Errorf("failed to update product file: %w", err)
	}
	
	return nil
}

// Delete removes a product
func (r *JSONProductRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if product exists
	if _, exists := r.cache[id]; !exists {
		return domain.ErrProductNotFound
	}
	
	// Delete from cache
	delete(r.cache, id)
	
	// Delete file
	filePath := filepath.Join(r.dataPath, fmt.Sprintf("%s.json", id))
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete product file: %w", err)
	}
	
	return nil
}

// FindByCategory retrieves products by category
func (r *JSONProductRepository) FindByCategory(ctx context.Context, category string) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	products := []domain.Product{}
	categoryLower := strings.ToLower(category)
	
	for _, product := range r.cache {
		if strings.ToLower(product.Category) == categoryLower {
			products = append(products, *product)
		}
	}
	
	return products, nil
}

// SearchProducts searches for products by name or description
func (r *JSONProductRepository) SearchProducts(ctx context.Context, query string) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	products := []domain.Product{}
	queryLower := strings.ToLower(query)
	
	for _, product := range r.cache {
		nameLower := strings.ToLower(product.Name)
		descLower := strings.ToLower(product.Description)
		
		if strings.Contains(nameLower, queryLower) || strings.Contains(descLower, queryLower) {
			products = append(products, *product)
		}
	}
	
	return products, nil
}

// Helper methods

func (r *JSONProductRepository) loadCache() error {
	files, err := ioutil.ReadDir(r.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Empty directory is OK
		}
		return err
	}
	
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		
		filePath := filepath.Join(r.dataPath, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file.Name(), err)
		}
		
		var product domain.Product
		if err := json.Unmarshal(data, &product); err != nil {
			return fmt.Errorf("failed to unmarshal product from %s: %w", file.Name(), err)
		}
		
		r.cache[product.ID] = &product
	}
	
	return nil
}

func (r *JSONProductRepository) saveToFile(product *domain.Product) error {
	filePath := filepath.Join(r.dataPath, fmt.Sprintf("%s.json", product.ID))
	
	data, err := json.MarshalIndent(product, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}
	
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

func (r *JSONProductRepository) initializeSampleData() {
	sampleProducts := []domain.Product{
		{
			ID:          "prod-001",
			Name:        "MacBook Pro 14\"",
			ImageURL:    "https://example.com/macbook-pro-14.jpg",
			Description: "Apple MacBook Pro 14-inch with M3 Pro chip",
			Price:       1999.99,
			Currency:    "USD",
			Rating:      4.8,
			Category:    "Laptops",
			Brand:       "Apple",
			Stock:       50,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Specifications: map[string]interface{}{
				"processor":    "Apple M3 Pro",
				"ram":          "18GB",
				"storage":      "512GB SSD",
				"display":      "14.2-inch Liquid Retina XDR",
				"battery_life": "17 hours",
				"weight":       "3.5 lbs",
			},
		},
		{
			ID:          "prod-002",
			Name:        "Dell XPS 13",
			ImageURL:    "https://example.com/dell-xps-13.jpg",
			Description: "Dell XPS 13 Ultra-thin laptop with Intel Core Ultra",
			Price:       1499.99,
			Currency:    "USD",
			Rating:      4.5,
			Category:    "Laptops",
			Brand:       "Dell",
			Stock:       75,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Specifications: map[string]interface{}{
				"processor":    "Intel Core Ultra 7",
				"ram":          "16GB",
				"storage":      "512GB SSD",
				"display":      "13.4-inch FHD+",
				"battery_life": "12 hours",
				"weight":       "2.6 lbs",
			},
		},
		{
			ID:          "prod-003",
			Name:        "ThinkPad X1 Carbon",
			ImageURL:    "https://example.com/thinkpad-x1.jpg",
			Description: "Lenovo ThinkPad X1 Carbon Gen 11 Business Laptop",
			Price:       1799.99,
			Currency:    "USD",
			Rating:      4.6,
			Category:    "Laptops",
			Brand:       "Lenovo",
			Stock:       40,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Specifications: map[string]interface{}{
				"processor":    "Intel Core i7-1365U",
				"ram":          "32GB",
				"storage":      "1TB SSD",
				"display":      "14-inch WUXGA",
				"battery_life": "15 hours",
				"weight":       "2.48 lbs",
			},
		},
		{
			ID:          "prod-004",
			Name:        "Surface Laptop 5",
			ImageURL:    "https://example.com/surface-laptop-5.jpg",
			Description: "Microsoft Surface Laptop 5 with touchscreen",
			Price:       1299.99,
			Currency:    "USD",
			Rating:      4.3,
			Category:    "Laptops",
			Brand:       "Microsoft",
			Stock:       60,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Specifications: map[string]interface{}{
				"processor":    "Intel Core i5-1235U",
				"ram":          "8GB",
				"storage":      "256GB SSD",
				"display":      "13.5-inch PixelSense",
				"battery_life": "18 hours",
				"weight":       "2.8 lbs",
			},
		},
		{
			ID:          "prod-005",
			Name:        "HP Spectre x360",
			ImageURL:    "https://example.com/hp-spectre.jpg",
			Description: "HP Spectre x360 2-in-1 Convertible Laptop",
			Price:       1599.99,
			Currency:    "USD",
			Rating:      4.4,
			Category:    "Laptops",
			Brand:       "HP",
			Stock:       35,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Specifications: map[string]interface{}{
				"processor":    "Intel Core i7-1355U",
				"ram":          "16GB",
				"storage":      "512GB SSD",
				"display":      "13.5-inch OLED",
				"battery_life": "13 hours",
				"weight":       "3.01 lbs",
			},
		},
	}
	
	for i := range sampleProducts {
		r.cache[sampleProducts[i].ID] = &sampleProducts[i]
		r.saveToFile(&sampleProducts[i])
	}
}
