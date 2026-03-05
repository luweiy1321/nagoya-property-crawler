package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Local database
	localDB, err := sql.Open("postgres", "host=localhost port=5432 user=lw dbname=nagoya_properties sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer localDB.Close()

	// Neon database connection string
	neonConn := "postgresql://neondb_owner:npg_UBEigRoV6Dk5@ep-floral-cherry-a1xz7gdk.ap-southeast-1.aws.neon.tech/neondb?sslmode=require"

	// Neon database
	neonDB, err := sql.Open("postgres", neonConn)
	if err != nil {
		log.Fatal(err)
	}
	defer neonDB.Close()

	// Test connections
	if err := localDB.Ping(); err != nil {
		log.Fatalf("Local DB error: %v", err)
	}
	fmt.Println("✓ Local database connected")

	if err := neonDB.Ping(); err != nil {
		log.Fatalf("Neon DB error: %v", err)
	}
	fmt.Println("✓ Neon database connected")

	// Create table in Neon - simpler version without JSONB for now
	fmt.Println("\nCreating tables in Neon...")
	createTableSQL := `
	DROP TABLE IF EXISTS properties CASCADE;

	CREATE TABLE properties (
		id SERIAL PRIMARY KEY,
		source VARCHAR(50) NOT NULL,
		property_id VARCHAR(100) NOT NULL,
		listing_type VARCHAR(20) NOT NULL,
		title TEXT,
		price INTEGER,
		price_display VARCHAR(100),
		address TEXT,
		area DECIMAL(10, 2),
		layout VARCHAR(20),
		floor VARCHAR(50),
		building_type VARCHAR(50),
		construction_year INTEGER,
		station_name VARCHAR(100),
		walking_minutes INTEGER,
		detail_url TEXT,
		scraped_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		CONSTRAINT unique_property UNIQUE (source, property_id)
	);

	CREATE INDEX idx_properties_source ON properties(source);
	CREATE INDEX idx_properties_listing_type ON properties(listing_type);
	`
	_, err = neonDB.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
	fmt.Println("✓ Tables created")

	// Fetch data from local database
	fmt.Println("\nFetching data from local database...")
	rows, err := localDB.Query(`
		SELECT source, property_id, listing_type, title, price, price_display,
			   address, area, layout, floor, building_type, construction_year,
			   station_name, walking_minutes, detail_url
		FROM properties
	`)
	if err != nil {
		log.Fatalf("Error querying local DB: %v", err)
	}
	defer rows.Close()

	// Prepare insert statement for Neon
	insertSQL := `
	INSERT INTO properties (
		source, property_id, listing_type, title, price, price_display,
		address, area, layout, floor, building_type, construction_year,
		station_name, walking_minutes, detail_url
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	ON CONFLICT (source, property_id) DO UPDATE SET
		title = EXCLUDED.title,
		price = EXCLUDED.price,
		price_display = EXCLUDED.price_display,
		address = EXCLUDED.address,
		area = EXCLUDED.area,
		layout = EXCLUDED.layout,
		floor = EXCLUDED.floor,
		station_name = EXCLUDED.station_name,
		walking_minutes = EXCLUDED.walking_minutes,
		detail_url = EXCLUDED.detail_url,
		updated_at = NOW()
	`

	stmt, err := neonDB.Prepare(insertSQL)
	if err != nil {
		log.Fatalf("Error preparing statement: %v", err)
	}
	defer stmt.Close()

	// Copy data
	count := 0
	for rows.Next() {
		var source, propertyID, listingType, title, priceDisplay, address, layout, floor, buildingType, stationName, detailURL sql.NullString
		var price, constructionYear, walkingMinutes sql.NullInt64
		var area sql.NullFloat64

		err = rows.Scan(
			&source, &propertyID, &listingType, &title, &price, &priceDisplay,
			&address, &area, &layout, &floor, &buildingType, &constructionYear,
			&stationName, &walkingMinutes, &detailURL,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		_, err = stmt.Exec(
			source.String, propertyID.String, listingType.String,
			nullToString(title),
			nullToInt64(price),
			nullToString(priceDisplay),
			nullToString(address),
			nullToFloat64(area),
			nullToString(layout),
			nullToString(floor),
			nullToString(buildingType),
			nullToInt64(constructionYear),
			nullToString(stationName),
			nullToInt32(walkingMinutes),
			nullToString(detailURL),
		)
		if err != nil {
			log.Printf("Error inserting %s: %v", propertyID.String, err)
			continue
		}

		count++
		if count%10 == 0 {
			fmt.Printf("  Copied %d records...\n", count)
		}
	}

	fmt.Printf("\n✅ Successfully copied %d records to Neon!\n", count)

	// Verify the data
	var verifyCount int
	neonDB.QueryRow("SELECT COUNT(*) FROM properties").Scan(&verifyCount)
	fmt.Printf("✓ Verified: %d records in Neon database\n", verifyCount)

	// Print connection info for Streamlit
	fmt.Println("\n=== Streamlit Cloud Secrets ===")
	fmt.Println("Copy these to your Streamlit app:")
	fmt.Println("```")
	fmt.Println("DB_HOST=ep-floral-cherry-a1xz7gdk.ap-southeast-1.aws.neon.tech")
	fmt.Println("DB_PORT=5432")
	fmt.Println("DB_USER=neondb_owner")
	fmt.Println("DB_PASSWORD=npg_UBEigRoV6Dk5")
	fmt.Println("DB_NAME=neondb")
	fmt.Println("```")
}

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullToInt64(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

func nullToFloat64(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}

func nullToInt32(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return int32(ni.Int64)
	}
	return nil
}
