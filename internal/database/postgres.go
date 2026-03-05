package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"nagoya-property-crawler/internal/config"
	"nagoya-property-crawler/internal/models"
)

// PostgresDB implements the models.Repository interface for PostgreSQL
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.DatabaseConfig) (*PostgresDB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.DBName, cfg.SSLMode,
	)
	if cfg.Password != "" {
		connStr = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
		)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{db: db}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// Insert inserts a new property into the database
func (p *PostgresDB) Insert(property *models.Property) error {
	query := `
		INSERT INTO properties (
			source, property_id, listing_type, title, price, price_display,
			address, area, layout, floor, building_type, construction_year,
			building_floors, station_name, walking_minutes, latitude, longitude,
			contact_name, contact_phone, contact_email, image_urls, detail_url,
			scraped_at, valid_until
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24
		) RETURNING id, updated_at
	`

	err := p.db.QueryRow(
		query,
		property.Source, property.PropertyID, property.ListingType,
		property.Title, property.Price, property.PriceDisplay,
		property.Address, property.Area, property.Layout,
		property.Floor, property.BuildingType, property.ConstructionYear,
		property.BuildingFloors, property.StationName, property.WalkingMinutes,
		property.Latitude, property.Longitude,
		property.ContactName, property.ContactPhone, property.ContactEmail,
		pq.Array(property.ImageURLs), property.DetailURL,
		property.ScrapedAt, property.ValidUntil,
	).Scan(&property.ID, &property.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert property: %w", err)
	}

	return nil
}

// Upsert inserts or updates a property based on unique constraint
func (p *PostgresDB) Upsert(property *models.Property) error {
	query := `
		INSERT INTO properties (
			source, property_id, listing_type, title, price, price_display,
			address, area, layout, floor, building_type, construction_year,
			building_floors, station_name, walking_minutes, latitude, longitude,
			contact_name, contact_phone, contact_email, image_urls, detail_url,
			scraped_at, valid_until
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24
		) ON CONFLICT (source, property_id)
		DO UPDATE SET
			title = EXCLUDED.title,
			price = EXCLUDED.price,
			price_display = EXCLUDED.price_display,
			address = EXCLUDED.address,
			area = EXCLUDED.area,
			layout = EXCLUDED.layout,
			floor = EXCLUDED.floor,
			building_type = EXCLUDED.building_type,
			construction_year = EXCLUDED.construction_year,
			building_floors = EXCLUDED.building_floors,
			station_name = EXCLUDED.station_name,
			walking_minutes = EXCLUDED.walking_minutes,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			contact_name = EXCLUDED.contact_name,
			contact_phone = EXCLUDED.contact_phone,
			contact_email = EXCLUDED.contact_email,
			image_urls = EXCLUDED.image_urls,
			detail_url = EXCLUDED.detail_url,
			scraped_at = EXCLUDED.scraped_at,
			valid_until = EXCLUDED.valid_until,
			updated_at = NOW()
		RETURNING id, updated_at
	`

	err := p.db.QueryRow(
		query,
		property.Source, property.PropertyID, property.ListingType,
		property.Title, property.Price, property.PriceDisplay,
		property.Address, property.Area, property.Layout,
		property.Floor, property.BuildingType, property.ConstructionYear,
		property.BuildingFloors, property.StationName, property.WalkingMinutes,
		property.Latitude, property.Longitude,
		property.ContactName, property.ContactPhone, property.ContactEmail,
		pq.Array(property.ImageURLs), property.DetailURL,
		property.ScrapedAt, property.ValidUntil,
	).Scan(&property.ID, &property.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert property: %w", err)
	}

	return nil
}

// GetByID retrieves a property by ID
func (p *PostgresDB) GetByID(id int64) (*models.Property, error) {
	query := `
		SELECT id, source, property_id, listing_type, title, price, price_display,
			address, area, layout, floor, building_type, construction_year,
			building_floors, station_name, walking_minutes, latitude, longitude,
			contact_name, contact_phone, contact_email, image_urls, detail_url,
			scraped_at, valid_until, updated_at
		FROM properties
		WHERE id = $1
	`

	property := &models.Property{}
	err := p.db.QueryRow(query, id).Scan(
		&property.ID, &property.Source, &property.PropertyID, &property.ListingType,
		&property.Title, &property.Price, &property.PriceDisplay,
		&property.Address, &property.Area, &property.Layout,
		&property.Floor, &property.BuildingType, &property.ConstructionYear,
		&property.BuildingFloors, &property.StationName, &property.WalkingMinutes,
		&property.Latitude, &property.Longitude,
		&property.ContactName, &property.ContactPhone, &property.ContactEmail,
		pq.Array(&property.ImageURLs), &property.DetailURL,
		&property.ScrapedAt, &property.ValidUntil, &property.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("property not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get property: %w", err)
	}

	return property, nil
}

// GetBySourceAndID retrieves a property by source and property_id
func (p *PostgresDB) GetBySourceAndID(source models.PropertySource, propertyID string) (*models.Property, error) {
	query := `
		SELECT id, source, property_id, listing_type, title, price, price_display,
			address, area, layout, floor, building_type, construction_year,
			building_floors, station_name, walking_minutes, latitude, longitude,
			contact_name, contact_phone, contact_email, image_urls, detail_url,
			scraped_at, valid_until, updated_at
		FROM properties
		WHERE source = $1 AND property_id = $2
	`

	property := &models.Property{}
	err := p.db.QueryRow(query, source, propertyID).Scan(
		&property.ID, &property.Source, &property.PropertyID, &property.ListingType,
		&property.Title, &property.Price, &property.PriceDisplay,
		&property.Address, &property.Area, &property.Layout,
		&property.Floor, &property.BuildingType, &property.ConstructionYear,
		&property.BuildingFloors, &property.StationName, &property.WalkingMinutes,
		&property.Latitude, &property.Longitude,
		&property.ContactName, &property.ContactPhone, &property.ContactEmail,
		pq.Array(&property.ImageURLs), &property.DetailURL,
		&property.ScrapedAt, &property.ValidUntil, &property.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("property not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get property: %w", err)
	}

	return property, nil
}

// List retrieves properties matching the given filters
func (p *PostgresDB) List(filter models.PropertyFilter) ([]*models.Property, error) {
	query := `
		SELECT id, source, property_id, listing_type, title, price, price_display,
			address, area, layout, floor, building_type, construction_year,
			building_floors, station_name, walking_minutes, latitude, longitude,
			contact_name, contact_phone, contact_email, image_urls, detail_url,
			scraped_at, valid_until, updated_at
		FROM properties
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if filter.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argCount)
		args = append(args, filter.Source)
		argCount++
	}

	if filter.ListingType != "" {
		query += fmt.Sprintf(" AND listing_type = $%d", argCount)
		args = append(args, filter.ListingType)
		argCount++
	}

	if filter.MinPrice > 0 {
		query += fmt.Sprintf(" AND price >= $%d", argCount)
		args = append(args, filter.MinPrice)
		argCount++
	}

	if filter.MaxPrice > 0 {
		query += fmt.Sprintf(" AND price <= $%d", argCount)
		args = append(args, filter.MaxPrice)
		argCount++
	}

	if filter.MinArea > 0 {
		query += fmt.Sprintf(" AND area >= $%d", argCount)
		args = append(args, filter.MinArea)
		argCount++
	}

	if filter.MaxArea > 0 {
		query += fmt.Sprintf(" AND area <= $%d", argCount)
		args = append(args, filter.MaxArea)
		argCount++
	}

	if filter.Layout != "" {
		query += fmt.Sprintf(" AND layout = $%d", argCount)
		args = append(args, filter.Layout)
		argCount++
	}

	if filter.StationName != "" {
		query += fmt.Sprintf(" AND station_name ILIKE $%d", argCount)
		args = append(args, "%"+filter.StationName+"%")
		argCount++
	}

	if filter.OnlyActive {
		query += fmt.Sprintf(" AND (valid_until IS NULL OR valid_until > CURRENT_DATE)")
	}

	query += " ORDER BY scraped_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list properties: %w", err)
	}
	defer rows.Close()

	var properties []*models.Property
	for rows.Next() {
		property := &models.Property{}
		err := rows.Scan(
			&property.ID, &property.Source, &property.PropertyID, &property.ListingType,
			&property.Title, &property.Price, &property.PriceDisplay,
			&property.Address, &property.Area, &property.Layout,
			&property.Floor, &property.BuildingType, &property.ConstructionYear,
			&property.BuildingFloors, &property.StationName, &property.WalkingMinutes,
			&property.Latitude, &property.Longitude,
			&property.ContactName, &property.ContactPhone, &property.ContactEmail,
			pq.Array(&property.ImageURLs), &property.DetailURL,
			&property.ScrapedAt, &property.ValidUntil, &property.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan property: %w", err)
		}
		properties = append(properties, property)
	}

	return properties, nil
}

// Count returns the count of properties matching the given filters
func (p *PostgresDB) Count(filter models.PropertyFilter) (int64, error) {
	query := "SELECT COUNT(*) FROM properties WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if filter.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argCount)
		args = append(args, filter.Source)
		argCount++
	}

	if filter.ListingType != "" {
		query += fmt.Sprintf(" AND listing_type = $%d", argCount)
		args = append(args, filter.ListingType)
		argCount++
	}

	if filter.MinPrice > 0 {
		query += fmt.Sprintf(" AND price >= $%d", argCount)
		args = append(args, filter.MinPrice)
		argCount++
	}

	if filter.MaxPrice > 0 {
		query += fmt.Sprintf(" AND price <= $%d", argCount)
		args = append(args, filter.MaxPrice)
		argCount++
	}

	if filter.MinArea > 0 {
		query += fmt.Sprintf(" AND area >= $%d", argCount)
		args = append(args, filter.MinArea)
		argCount++
	}

	if filter.MaxArea > 0 {
		query += fmt.Sprintf(" AND area <= $%d", argCount)
		args = append(args, filter.MaxArea)
		argCount++
	}

	if filter.Layout != "" {
		query += fmt.Sprintf(" AND layout = $%d", argCount)
		args = append(args, filter.Layout)
		argCount++
	}

	if filter.OnlyActive {
		query += " AND (valid_until IS NULL OR valid_until > CURRENT_DATE)"
	}

	var count int64
	err := p.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count properties: %w", err)
	}

	return count, nil
}

// GetStats returns statistics grouped by source and listing type
func (p *PostgresDB) GetStats() ([]models.PropertyStats, error) {
	query := `
		SELECT
			source,
			listing_type,
			COUNT(*) as total_count,
			COALESCE(AVG(price), 0) as avg_price,
			COALESCE(MIN(price), 0) as min_price,
			COALESCE(MAX(price), 0) as max_price,
			COALESCE(AVG(area), 0) as avg_area
		FROM properties
		WHERE valid_until IS NULL OR valid_until > CURRENT_DATE
		GROUP BY source, listing_type
		ORDER BY source, listing_type
	`

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer rows.Close()

	var stats []models.PropertyStats
	for rows.Next() {
		var stat models.PropertyStats
		err := rows.Scan(
			&stat.Source,
			&stat.ListingType,
			&stat.TotalCount,
			&stat.AvgPrice,
			&stat.MinPrice,
			&stat.MaxPrice,
			&stat.AvgArea,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// DeleteExpired removes properties past their valid_until date
func (p *PostgresDB) DeleteExpired() (int64, error) {
	query := `DELETE FROM properties WHERE valid_until IS NOT NULL AND valid_until <= CURRENT_DATE`

	result, err := p.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired properties: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return count, nil
}

// DeleteBySource removes all properties from a specific source
func (p *PostgresDB) DeleteBySource(source models.PropertySource) (int64, error) {
	query := `DELETE FROM properties WHERE source = $1`

	result, err := p.db.Exec(query, source)
	if err != nil {
		return 0, fmt.Errorf("failed to delete properties: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return count, nil
}

// UpdateValidUntil updates the expiration date for a property
func (p *PostgresDB) UpdateValidUntil(id int64, validUntil time.Time) error {
	query := `UPDATE properties SET valid_until = $1 WHERE id = $2`

	_, err := p.db.Exec(query, validUntil, id)
	if err != nil {
		return fmt.Errorf("failed to update valid_until: %w", err)
	}

	return nil
}

// Health checks the database connection health
func (p *PostgresDB) Health(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// BeginTx starts a new transaction
func (p *PostgresDB) BeginTx() (*sql.Tx, error) {
	return p.db.Begin()
}

// WithTx executes a function within a transaction
func (p *PostgresDB) WithTx(fn func(*sql.Tx) error) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("exec failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// UpsertBatch inserts or updates multiple properties in a single transaction
func (p *PostgresDB) UpsertBatch(properties []*models.Property) error {
	return p.WithTx(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`
			INSERT INTO properties (
				source, property_id, listing_type, title, price, price_display,
				address, area, layout, floor, building_type, construction_year,
				building_floors, station_name, walking_minutes, latitude, longitude,
				contact_name, contact_phone, contact_email, image_urls, detail_url,
				scraped_at, valid_until
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
				$16, $17, $18, $19, $20, $21, $22, $23, $24
			) ON CONFLICT (source, property_id)
			DO UPDATE SET
				title = EXCLUDED.title,
				price = EXCLUDED.price,
				price_display = EXCLUDED.price_display,
				address = EXCLUDED.address,
				area = EXCLUDED.area,
				layout = EXCLUDED.layout,
				floor = EXCLUDED.floor,
				building_type = EXCLUDED.building_type,
				construction_year = EXCLUDED.construction_year,
				building_floors = EXCLUDED.building_floors,
				station_name = EXCLUDED.station_name,
				walking_minutes = EXCLUDED.walking_minutes,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude,
				contact_name = EXCLUDED.contact_name,
				contact_phone = EXCLUDED.contact_phone,
				contact_email = EXCLUDED.contact_email,
				image_urls = EXCLUDED.image_urls,
				detail_url = EXCLUDED.detail_url,
				scraped_at = EXCLUDED.scraped_at,
				valid_until = EXCLUDED.valid_until,
				updated_at = NOW()
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, property := range properties {
			_, err := stmt.Exec(
				property.Source, property.PropertyID, property.ListingType,
				property.Title, property.Price, property.PriceDisplay,
				property.Address, property.Area, property.Layout,
				property.Floor, property.BuildingType, property.ConstructionYear,
				property.BuildingFloors, property.StationName, property.WalkingMinutes,
				property.Latitude, property.Longitude,
				property.ContactName, property.ContactPhone, property.ContactEmail,
				pq.Array(property.ImageURLs), property.DetailURL,
				property.ScrapedAt, property.ValidUntil,
			)
			if err != nil {
				return fmt.Errorf("failed to execute statement: %w", err)
			}
		}

		return nil
	})
}
