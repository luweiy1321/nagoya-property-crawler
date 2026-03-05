package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ListingType represents the type of property listing
type ListingType string

const (
	ListingTypeSale         ListingType = "sale"          // 出售
	ListingTypeRent         ListingType = "rent"          // 出租
	ListingTypeBuyRequest   ListingType = "buy_request"   // 购买请求
	ListingTypeRentRequest  ListingType = "rent_request"  // 租赁请求
)

// PropertySource represents the data source
type PropertySource string

const (
	SourceSUUMO  PropertySource = "suumo"
	SourceHOMES  PropertySource = "homes"
	SourceAtHome PropertySource = "athome"
)

// StringArray is a custom type for handling string arrays in PostgreSQL
type StringArray []string

// Scan implements the sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

// Value implements the driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Property represents a real estate property listing
type Property struct {
	ID        int64          `json:"id" db:"id"`
	Source    PropertySource `json:"source" db:"source"`
	PropertyID string        `json:"property_id" db:"property_id"`
	ListingType ListingType  `json:"listing_type" db:"listing_type"`

	// Basic information
	Title        string  `json:"title" db:"title"`
	Price        int     `json:"price" db:"price"`
	PriceDisplay string  `json:"price_display" db:"price_display"`
	Address      string  `json:"address" db:"address"`
	Area         float64 `json:"area" db:"area"`
	Layout       string  `json:"layout" db:"layout"`

	// Detailed information
	Floor            string  `json:"floor" db:"floor"`
	BuildingType     string  `json:"building_type" db:"building_type"`
	ConstructionYear int     `json:"construction_year" db:"construction_year"`
	BuildingFloors   int     `json:"building_floors" db:"building_floors"`

	// Location information
	StationName    string  `json:"station_name" db:"station_name"`
	WalkingMinutes int     `json:"walking_minutes" db:"walking_minutes"`
	Latitude       float64 `json:"latitude" db:"latitude"`
	Longitude      float64 `json:"longitude" db:"longitude"`

	// Contact information
	ContactName  string `json:"contact_name" db:"contact_name"`
	ContactPhone string `json:"contact_phone" db:"contact_phone"`
	ContactEmail string `json:"contact_email" db:"contact_email"`

	// Images and links
	ImageURLs StringArray `json:"image_urls" db:"image_urls"`
	DetailURL string      `json:"detail_url" db:"detail_url"`

	// Metadata
	ScrapedAt  time.Time    `json:"scraped_at" db:"scraped_at"`
	ValidUntil *time.Time   `json:"valid_until" db:"valid_until"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at"`
}

// PropertyFilter represents filter criteria for property queries
type PropertyFilter struct {
	Source      PropertySource `json:"source"`
	ListingType ListingType    `json:"listing_type"`
	MinPrice    int            `json:"min_price"`
	MaxPrice    int            `json:"max_price"`
	MinArea     float64        `json:"min_area"`
	MaxArea     float64        `json:"max_area"`
	Layout      string         `json:"layout"`
	StationName string         `json:"station_name"`
	Limit       int            `json:"limit"`
	Offset      int            `json:"offset"`
	OnlyActive  bool           `json:"only_active"`
}

// PropertyStats represents statistics about properties
type PropertyStats struct {
	Source      PropertySource `json:"source"`
	ListingType ListingType    `json:"listing_type"`
	TotalCount  int64          `json:"total_count"`
	AvgPrice    float64        `json:"avg_price"`
	MinPrice    int            `json:"min_price"`
	MaxPrice    int            `json:"max_price"`
	AvgArea     float64        `json:"avg_area"`
}

// Repository defines the interface for property data operations
type Repository interface {
	// Insert inserts a new property
	Insert(property *Property) error

	// Upsert inserts or updates a property based on unique constraint
	Upsert(property *Property) error

	// GetByID retrieves a property by ID
	GetByID(id int64) (*Property, error)

	// GetBySourceAndID retrieves a property by source and property_id
	GetBySourceAndID(source PropertySource, propertyID string) (*Property, error)

	// List retrieves properties matching the given filters
	List(filter PropertyFilter) ([]*Property, error)

	// Count returns the count of properties matching the given filters
	Count(filter PropertyFilter) (int64, error)

	// GetStats returns statistics grouped by source and listing type
	GetStats() ([]PropertyStats, error)

	// DeleteExpired removes properties past their valid_until date
	DeleteExpired() (int64, error)

	// DeleteBySource removes all properties from a specific source
	DeleteBySource(source PropertySource) (int64, error)

	// UpdateValidUntil updates the expiration date for a property
	UpdateValidUntil(id int64, validUntil time.Time) error
}

// NewProperty creates a new Property with default values
func NewProperty(source PropertySource, propertyID string, listingType ListingType) *Property {
	now := time.Now()
	return &Property{
		Source:      source,
		PropertyID:  propertyID,
		ListingType: listingType,
		ScrapedAt:   now,
		UpdatedAt:   now,
	}
}

// IsExpired checks if the property listing has expired
func (p *Property) IsExpired() bool {
	if p.ValidUntil == nil {
		return false
	}
	return p.ValidUntil.Before(time.Now())
}

// GetPriceInMan returns the price in 万 (man) units
func (p *Property) GetPriceInMan() float64 {
	return float64(p.Price) / 10000
}

// GetPriceInYen returns formatted price in Japanese yen
func (p *Property) GetPriceInYen() string {
	if p.PriceDisplay != "" {
		return p.PriceDisplay
	}
	return fmt.Sprintf("%d円", p.Price)
}
