-- Nagoya Property Crawler Database Schema
-- PostgreSQL initialization script

-- Enable UUID extension (optional, for future use)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Properties table
CREATE TABLE IF NOT EXISTS properties (
    id SERIAL PRIMARY KEY,

    -- Source identification
    source VARCHAR(50) NOT NULL,           -- suumo, homes, athome
    property_id VARCHAR(100) NOT NULL,     -- Website's property ID
    listing_type VARCHAR(20) NOT NULL,     -- sale, rent, buy_request, rent_request

    -- Basic information
    title TEXT,
    price INTEGER,                         -- Price in JPY
    price_display VARCHAR(100),            -- Display price (e.g., "1000万円")
    address TEXT,
    area DECIMAL(10, 2),                   -- Area in square meters
    layout VARCHAR(20),                    -- Room layout (1K, 1LDK, etc.)

    -- Detailed information
    floor VARCHAR(50),                     -- Floor number
    building_type VARCHAR(50),             -- Building type (mansion, apartment, house)
    construction_year INTEGER,             -- Year built
    building_floors INTEGER,               -- Total floors in building

    -- Location information
    station_name VARCHAR(100),             -- Nearest station
    walking_minutes INTEGER,               -- Walking distance to station (minutes)
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),

    -- Contact information
    contact_name VARCHAR(200),
    contact_phone VARCHAR(50),
    contact_email VARCHAR(200),

    -- Images and links
    image_urls JSONB,                      -- Array of image URLs
    detail_url TEXT,                       -- Detail page URL

    -- Metadata
    scraped_at TIMESTAMP DEFAULT NOW(),
    valid_until DATE,                      -- Listing expiration date
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_property UNIQUE (source, property_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_properties_source ON properties(source);
CREATE INDEX IF NOT EXISTS idx_properties_listing_type ON properties(listing_type);
CREATE INDEX IF NOT EXISTS idx_properties_price ON properties(price);
CREATE INDEX IF NOT EXISTS idx_properties_area ON properties(area);
CREATE INDEX IF NOT EXISTS idx_properties_scraped_at ON properties(scraped_at);
CREATE INDEX IF NOT EXISTS idx_properties_valid_until ON properties(valid_until);
CREATE INDEX IF NOT EXISTS idx_properties_source_property_id ON properties(source, property_id);

-- Index for text search
CREATE INDEX IF NOT EXISTS idx_properties_title_gin ON properties USING gin(to_tsvector('japanese', title));
CREATE INDEX IF NOT EXISTS idx_properties_address_gin ON properties USING gin(to_tsvector('japanese', address));

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-update updated_at
DROP TRIGGER IF EXISTS update_properties_updated_at ON properties;
CREATE TRIGGER update_properties_updated_at
    BEFORE UPDATE ON properties
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- View for active listings (not expired)
CREATE OR REPLACE VIEW active_properties AS
SELECT * FROM properties
WHERE valid_until IS NULL OR valid_until > CURRENT_DATE;

-- View for statistics
CREATE OR REPLACE VIEW property_stats AS
SELECT
    source,
    listing_type,
    COUNT(*) as total_count,
    AVG(price) as avg_price,
    MIN(price) as min_price,
    MAX(price) as max_price,
    AVG(area) as avg_area
FROM properties
WHERE valid_until IS NULL OR valid_until > CURRENT_DATE
GROUP BY source, listing_type;

-- Comment on table
COMMENT ON TABLE properties IS 'Japanese property listings from SUUMO, HOMES, and at-home websites';
COMMENT ON COLUMN properties.source IS 'Data source: suumo, homes, or athome';
COMMENT ON COLUMN properties.property_id IS 'Unique identifier from the source website';
COMMENT ON COLUMN properties.listing_type IS 'Type of listing: sale, rent, buy_request, rent_request';
COMMENT ON COLUMN properties.price IS 'Price in Japanese Yen (integer)';
COMMENT ON COLUMN properties.price_display IS 'Human-readable price string (e.g., 1,000万円)';
COMMENT ON COLUMN properties.image_urls IS 'JSONB array of image URLs';
