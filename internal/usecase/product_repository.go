package usecase

import (
	"context"
	"github.com/javinez/product-comparison-api/internal/domain"
)

// ProductRepository defines the interface for product data persistence
type ProductRepository interface {
	// FindByID retrieves a single product by its ID
	FindByID(ctx context.Context, id string) (*domain.Product, error)
	
	// FindByIDs retrieves multiple products by their IDs
	FindByIDs(ctx context.Context, ids []string) ([]domain.Product, error)
	
	// FindAll retrieves all products with optional pagination
	FindAll(ctx context.Context, limit, offset int) ([]domain.Product, error)
	
	// Save persists a product
	Save(ctx context.Context, product *domain.Product) error
	
	// Update updates an existing product
	Update(ctx context.Context, product *domain.Product) error
	
	// Delete removes a product by its ID
	Delete(ctx context.Context, id string) error
	
	// FindByCategory retrieves products by category
	FindByCategory(ctx context.Context, category string) ([]domain.Product, error)
	
	// SearchProducts searches products by name or description
	SearchProducts(ctx context.Context, query string) ([]domain.Product, error)
}
