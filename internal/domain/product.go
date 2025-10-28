package domain

import (
	"errors"
	"time"
)

// Product represents a product in our domain
type Product struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	ImageURL      string                 `json:"image_url"`
	Description   string                 `json:"description"`
	Price         float64                `json:"price"`
	Currency      string                 `json:"currency"`
	Rating        float64                `json:"rating"`
	Specifications map[string]interface{} `json:"specifications"`
	Category      string                 `json:"category"`
	Brand         string                 `json:"brand"`
	Stock         int                    `json:"stock"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ProductComparison represents a comparison between multiple products
type ProductComparison struct {
	Products      []Product              `json:"products"`
	ComparisonID  string                 `json:"comparison_id"`
	CreatedAt     time.Time              `json:"created_at"`
	Differences   map[string][]string    `json:"differences"`
	Similarities  map[string]interface{} `json:"similarities"`
}

// Validation errors
var (
	ErrInvalidProductID   = errors.New("invalid product ID")
	ErrProductNotFound    = errors.New("product not found")
	ErrInvalidPrice       = errors.New("invalid product price")
	ErrInvalidRating      = errors.New("rating must be between 0 and 5")
	ErrEmptyProductName   = errors.New("product name cannot be empty")
	ErrInvalidComparison  = errors.New("comparison requires at least 2 products")
)

// Validate performs validation on the Product entity
func (p *Product) Validate() error {
	if p.ID == "" {
		return ErrInvalidProductID
	}
	if p.Name == "" {
		return ErrEmptyProductName
	}
	if p.Price < 0 {
		return ErrInvalidPrice
	}
	if p.Rating < 0 || p.Rating > 5 {
		return ErrInvalidRating
	}
	return nil
}

// CompareWith compares this product with another product
func (p *Product) CompareWith(other *Product) map[string]interface{} {
	comparison := make(map[string]interface{})
	
	comparison["price_difference"] = p.Price - other.Price
	comparison["rating_difference"] = p.Rating - other.Rating
	comparison["same_brand"] = p.Brand == other.Brand
	comparison["same_category"] = p.Category == other.Category
	
	return comparison
}
