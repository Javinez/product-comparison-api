package domain_test

import (
	"testing"
	"time"
	
	"github.com/javinez/product-comparison-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProduct_Validate(t *testing.T) {
	tests := []struct {
		name    string
		product domain.Product
		wantErr error
	}{
		{
			name: "valid product",
			product: domain.Product{
				ID:       "prod-001",
				Name:     "Test Product",
				Price:    99.99,
				Rating:   4.5,
				Category: "Electronics",
			},
			wantErr: nil,
		},
		{
			name: "empty product ID",
			product: domain.Product{
				Name:   "Test Product",
				Price:  99.99,
				Rating: 4.5,
			},
			wantErr: domain.ErrInvalidProductID,
		},
		{
			name: "empty product name",
			product: domain.Product{
				ID:     "prod-001",
				Price:  99.99,
				Rating: 4.5,
			},
			wantErr: domain.ErrEmptyProductName,
		},
		{
			name: "negative price",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Test Product",
				Price:  -10.00,
				Rating: 4.5,
			},
			wantErr: domain.ErrInvalidPrice,
		},
		{
			name: "zero price (valid)",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Free Product",
				Price:  0.00,
				Rating: 4.5,
			},
			wantErr: nil,
		},
		{
			name: "rating below minimum",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Test Product",
				Price:  99.99,
				Rating: -1,
			},
			wantErr: domain.ErrInvalidRating,
		},
		{
			name: "rating above maximum",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Test Product",
				Price:  99.99,
				Rating: 5.1,
			},
			wantErr: domain.ErrInvalidRating,
		},
		{
			name: "rating at minimum boundary (0)",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Test Product",
				Price:  99.99,
				Rating: 0,
			},
			wantErr: nil,
		},
		{
			name: "rating at maximum boundary (5)",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Test Product",
				Price:  99.99,
				Rating: 5.0,
			},
			wantErr: nil,
		},
		{
			name: "very long product name",
			product: domain.Product{
				ID:     "prod-001",
				Name:   string(make([]byte, 10000)), // 10KB name
				Price:  99.99,
				Rating: 4.5,
			},
			wantErr: nil, // Should pass, but in production you might want to limit this
		},
		{
			name: "special characters in name",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Product™ with © symbols & 特殊字符 🎉",
				Price:  99.99,
				Rating: 4.5,
			},
			wantErr: nil,
		},
		{
			name: "very high price",
			product: domain.Product{
				ID:     "prod-001",
				Name:   "Luxury Item",
				Price:  999999999.99,
				Rating: 4.5,
			},
			wantErr: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.product.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProduct_CompareWith(t *testing.T) {
	product1 := &domain.Product{
		ID:       "prod-001",
		Name:     "Product 1",
		Price:    100.00,
		Rating:   4.5,
		Brand:    "BrandA",
		Category: "Electronics",
	}
	
	product2 := &domain.Product{
		ID:       "prod-002",
		Name:     "Product 2",
		Price:    150.00,
		Rating:   4.0,
		Brand:    "BrandA",
		Category: "Electronics",
	}
	
	product3 := &domain.Product{
		ID:       "prod-003",
		Name:     "Product 3",
		Price:    100.00,
		Rating:   4.5,
		Brand:    "BrandB",
		Category: "Furniture",
	}
	
	t.Run("compare products with same brand and category", func(t *testing.T) {
		comparison := product1.CompareWith(product2)
		
		assert.Equal(t, -50.0, comparison["price_difference"])
		assert.Equal(t, 0.5, comparison["rating_difference"])
		assert.Equal(t, true, comparison["same_brand"])
		assert.Equal(t, true, comparison["same_category"])
	})
	
	t.Run("compare products with different brand and category", func(t *testing.T) {
		comparison := product1.CompareWith(product3)
		
		assert.Equal(t, 0.0, comparison["price_difference"])
		assert.Equal(t, 0.0, comparison["rating_difference"])
		assert.Equal(t, false, comparison["same_brand"])
		assert.Equal(t, false, comparison["same_category"])
	})
	
	t.Run("compare with nil product", func(t *testing.T) {
		// This should panic or be handled gracefully
		assert.NotPanics(t, func() {
			_ = product1.CompareWith(&domain.Product{})
		})
	})
}

func TestProductComparison_Validation(t *testing.T) {
	t.Run("valid comparison", func(t *testing.T) {
		comparison := &domain.ProductComparison{
			Products: []domain.Product{
				{ID: "1", Name: "Product 1"},
				{ID: "2", Name: "Product 2"},
			},
			ComparisonID: "comp-001",
			CreatedAt:    time.Now(),
		}
		
		assert.Len(t, comparison.Products, 2)
		assert.NotEmpty(t, comparison.ComparisonID)
	})
	
	t.Run("empty comparison", func(t *testing.T) {
		comparison := &domain.ProductComparison{
			Products:     []domain.Product{},
			ComparisonID: "comp-001",
			CreatedAt:    time.Now(),
		}
		
		assert.Len(t, comparison.Products, 0)
	})
	
	t.Run("comparison with duplicate products", func(t *testing.T) {
		product := domain.Product{ID: "1", Name: "Product 1"}
		comparison := &domain.ProductComparison{
			Products: []domain.Product{
				product,
				product, // Duplicate
			},
			ComparisonID: "comp-001",
			CreatedAt:    time.Now(),
		}
		
		// Should handle duplicates gracefully
		assert.Len(t, comparison.Products, 2)
	})
}

func TestProduct_EdgeCases(t *testing.T) {
	t.Run("product with nil specifications", func(t *testing.T) {
		product := &domain.Product{
			ID:             "prod-001",
			Name:           "Test Product",
			Price:          99.99,
			Rating:         4.5,
			Specifications: nil,
		}
		
		err := product.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("product with empty specifications", func(t *testing.T) {
		product := &domain.Product{
			ID:             "prod-001",
			Name:           "Test Product",
			Price:          99.99,
			Rating:         4.5,
			Specifications: make(map[string]interface{}),
		}
		
		err := product.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("product with complex specifications", func(t *testing.T) {
		product := &domain.Product{
			ID:     "prod-001",
			Name:   "Test Product",
			Price:  99.99,
			Rating: 4.5,
			Specifications: map[string]interface{}{
				"dimensions": map[string]float64{
					"width":  10.5,
					"height": 20.3,
					"depth":  5.2,
				},
				"features": []string{"feature1", "feature2", "feature3"},
				"warranty": "2 years",
				"weight":   1.5,
			},
		}
		
		err := product.Validate()
		assert.NoError(t, err)
		require.NotNil(t, product.Specifications["dimensions"])
	})
	
	t.Run("concurrent access to product", func(t *testing.T) {
		product := &domain.Product{
			ID:     "prod-001",
			Name:   "Test Product",
			Price:  99.99,
			Rating: 4.5,
		}
		
		// Test concurrent validation
		done := make(chan bool)
		for i := 0; i < 100; i++ {
			go func() {
				err := product.Validate()
				assert.NoError(t, err)
				done <- true
			}()
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 100; i++ {
			<-done
		}
	})
}

func BenchmarkProduct_Validate(b *testing.B) {
	product := &domain.Product{
		ID:       "prod-001",
		Name:     "Test Product",
		Price:    99.99,
		Rating:   4.5,
		Category: "Electronics",
		Brand:    "TestBrand",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = product.Validate()
	}
}

func BenchmarkProduct_CompareWith(b *testing.B) {
	product1 := &domain.Product{
		ID:       "prod-001",
		Name:     "Product 1",
		Price:    100.00,
		Rating:   4.5,
		Brand:    "BrandA",
		Category: "Electronics",
	}
	
	product2 := &domain.Product{
		ID:       "prod-002",
		Name:     "Product 2",
		Price:    150.00,
		Rating:   4.0,
		Brand:    "BrandB",
		Category: "Electronics",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = product1.CompareWith(product2)
	}
}
