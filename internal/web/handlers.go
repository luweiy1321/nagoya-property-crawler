package web

import (
	"encoding/csv"
	"encoding/json"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"nagoya-property-crawler/internal/database"
	"nagoya-property-crawler/internal/models"
)

//go:embed templates/*
var templates embed.FS

// Template functions
var templateFuncs = template.FuncMap{
	"div": func(a, b interface{}) float64 {
		var aVal, bVal float64
		switch v := a.(type) {
		case int:
			aVal = float64(v)
		case int64:
			aVal = float64(v)
		case float64:
			aVal = v
		}
		switch v := b.(type) {
		case int:
			bVal = float64(v)
		case int64:
			bVal = float64(v)
		case float64:
			bVal = v
		}
		if bVal == 0 {
			return 0
		}
		return aVal / bVal
	},
	"sub": func(a, b int) int {
		return a - b
	},
	"add": func(a, b int) int {
		return a + b
	},
	"iterate": func(count int) []int {
		var i []int
		for j := 1; j <= count; j++ {
			i = append(i, j)
		}
		return i
	},
	"preserveFilters": func() string {
		return "" // Can be enhanced to preserve filters
	},
	"gt": func(a, b interface{}) bool {
		return compareFloat(a, b) > 0
	},
	"le": func(a, b interface{}) bool {
		return compareFloat(a, b) <= 0
	},
	"eq": func(a, b interface{}) bool {
		return compareFloat(a, b) == 0
	},
}

func compareFloat(a, b interface{}) float64 {
	var aVal, bVal float64
	switch v := a.(type) {
	case int:
		aVal = float64(v)
	case int64:
		aVal = float64(v)
	case float64:
		aVal = v
	}
	switch v := b.(type) {
	case int:
		bVal = float64(v)
	case int64:
		bVal = float64(v)
	case float64:
		bVal = v
	}
	return aVal - bVal
}

// Server represents the web server
type Server struct {
	db        *database.PostgresDB
	templates *template.Template
}

// NewServer creates a new web server
func NewServer(db *database.PostgresDB) (*Server, error) {
	// Parse templates with functions
	tmpl, err := template.New("").Funcs(templateFuncs).ParseFS(templates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		db:        db,
		templates: tmpl,
	}, nil
}

// PageData represents common data for all pages
type PageData struct {
	Title        string
	Stats        []models.PropertyStats
	TotalCount   int64
	CurrentYear  int
	ErrorMessage string
}

// ListPageData represents data for the property list page
type ListPageData struct {
	PageData
	Properties  []*models.Property
	TotalCount  int64
	CurrentPage int
	TotalPages  int
	Filters     models.PropertyFilter
}

// DetailPageData represents data for the property detail page
type DetailPageData struct {
	PageData
	Property *models.Property
}

// HomeHandler renders the home page with statistics
func (s *Server) HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Get statistics
	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get total count
	totalCount, err := s.db.Count(models.PropertyFilter{OnlyActive: true})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:      "名古屋房产信息 - 首页",
		Stats:      stats,
		TotalCount: totalCount,
		CurrentYear: time.Now().Year(),
	}

	if err := s.templates.ExecuteTemplate(w, "home.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ListHandler renders the property list page with optional filters
func (s *Server) ListHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filters
	filter := s.parseFilters(r)

	// Default pagination
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	// Get properties
	properties, err := s.db.List(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	countFilter := filter
	countFilter.Limit = 0
	countFilter.Offset = 0
	totalCount, err := s.db.Count(countFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get statistics for sidebar
	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate total pages
	totalPages := int(totalCount) / filter.Limit
	if int(totalCount)%filter.Limit > 0 {
		totalPages++
	}

	currentPage := 1
	if filter.Offset > 0 {
		currentPage = filter.Offset/filter.Limit + 1
	}

	data := ListPageData{
		PageData: PageData{
			Title:       "房产列表",
			Stats:       stats,
			TotalCount:  totalCount,
			CurrentYear: time.Now().Year(),
		},
		Properties:  properties,
		TotalCount:  totalCount,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		Filters:     filter,
	}

	if err := s.templates.ExecuteTemplate(w, "list.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DetailHandler renders the property detail page
func (s *Server) DetailHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID from URL path
	// URL format: /detail/{id}
	path := r.URL.Path
	if len(path) <= len("/detail/") {
		http.Error(w, "Invalid property ID", http.StatusBadRequest)
		return
	}
	idStr := path[len("/detail/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid property ID", http.StatusBadRequest)
		return
	}

	// Get property
	property, err := s.db.GetByID(id)
	if err != nil {
		http.Error(w, "Property not found", http.StatusNotFound)
		return
	}

	// Get statistics
	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := DetailPageData{
		PageData: PageData{
			Title:       property.Title + " - 详情",
			Stats:       stats,
			TotalCount:  0,
			CurrentYear: time.Now().Year(),
		},
		Property: property,
	}

	if err := s.templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DownloadHandler handles CSV/JSON data export
func (s *Server) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	// Parse filters
	filter := s.parseFilters(r)

	// Get all matching properties (no limit for download)
	filter.Limit = 0
	filter.Offset = 0

	properties, err := s.db.List(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch format {
	case "json":
		s.downloadJSON(w, properties)
	case "csv":
		s.downloadCSV(w, properties)
	default:
		http.Error(w, "Invalid format. Use 'csv' or 'json'", http.StatusBadRequest)
	}
}

// APIStatsHandler returns statistics as JSON
func (s *Server) APIStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// APIPropertiesHandler returns properties as JSON with pagination
func (s *Server) APIPropertiesHandler(w http.ResponseWriter, r *http.Request) {
	filter := s.parseFilters(r)

	// Default pagination
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100 // Max 100 per page
	}

	properties, err := s.db.List(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCount, err := s.db.Count(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data":        properties,
		"total":       totalCount,
		"limit":       filter.Limit,
		"offset":      filter.Offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HealthHandler returns health status
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := s.db.Health(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status": "unhealthy", "error": "` + err.Error() + `"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy"}`))
}

// parseFilters parses query parameters into a PropertyFilter
func (s *Server) parseFilters(r *http.Request) models.PropertyFilter {
	filter := models.PropertyFilter{
		OnlyActive: true, // Default to only active listings
	}

	// Parse source
	if source := r.URL.Query().Get("source"); source != "" {
		filter.Source = models.PropertySource(source)
	}

	// Parse listing type
	if listingType := r.URL.Query().Get("type"); listingType != "" {
		filter.ListingType = models.ListingType(listingType)
	}

	// Parse price range
	if minPrice := r.URL.Query().Get("min_price"); minPrice != "" {
		if val, err := strconv.Atoi(minPrice); err == nil {
			filter.MinPrice = val
		}
	}
	if maxPrice := r.URL.Query().Get("max_price"); maxPrice != "" {
		if val, err := strconv.Atoi(maxPrice); err == nil {
			filter.MaxPrice = val
		}
	}

	// Parse area range
	if minArea := r.URL.Query().Get("min_area"); minArea != "" {
		if val, err := strconv.ParseFloat(minArea, 64); err == nil {
			filter.MinArea = val
		}
	}
	if maxArea := r.URL.Query().Get("max_area"); maxArea != "" {
		if val, err := strconv.ParseFloat(maxArea, 64); err == nil {
			filter.MaxArea = val
		}
	}

	// Parse layout
	if layout := r.URL.Query().Get("layout"); layout != "" {
		filter.Layout = layout
	}

	// Parse station
	if station := r.URL.Query().Get("station"); station != "" {
		filter.StationName = station
	}

	// Parse pagination
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			filter.Limit = val
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil {
			filter.Offset = val
		}
	}
	if page := r.URL.Query().Get("page"); page != "" {
		if val, err := strconv.Atoi(page); err == nil && val > 0 {
			limit := filter.Limit
			if limit == 0 {
				limit = 20
			}
			filter.Offset = (val - 1) * limit
		}
	}

	return filter
}

// downloadJSON sends properties as JSON file
func (s *Server) downloadJSON(w http.ResponseWriter, properties []*models.Property) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nagoya_properties_%s.json", time.Now().Format("20060102_150405")))

	json.NewEncoder(w).Encode(properties)
}

// downloadCSV sends properties as CSV file
func (s *Server) downloadCSV(w http.ResponseWriter, properties []*models.Property) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=nagoya_properties_%s.csv", time.Now().Format("20060102_150405")))

	// Write UTF-8 BOM for Excel compatibility
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID", "来源", "房源ID", "类型", "标题", "价格(日元)", "价格显示", "地址",
		"面积(㎡)", "房型", "楼层", "建筑类型", "建造年份", "总楼层",
		"最近车站", "步行分钟", "联系人", "电话", "详情链接", "爬取时间",
	}
	writer.Write(header)

	// Write data rows
	for _, p := range properties {
		record := []string{
			strconv.FormatInt(p.ID, 10),
			string(p.Source),
			p.PropertyID,
			string(p.ListingType),
			p.Title,
			strconv.Itoa(p.Price),
			p.PriceDisplay,
			p.Address,
			fmt.Sprintf("%.2f", p.Area),
			p.Layout,
			p.Floor,
			p.BuildingType,
			strconv.Itoa(p.ConstructionYear),
			strconv.Itoa(p.BuildingFloors),
			p.StationName,
			strconv.Itoa(p.WalkingMinutes),
			p.ContactName,
			p.ContactPhone,
			p.DetailURL,
			p.ScrapedAt.Format("2006-01-02 15:04:05"),
		}
		writer.Write(record)
	}
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Page routes (no method spec to use traditional ServeMux behavior)
	mux.HandleFunc("/list", s.ListHandler)
	mux.HandleFunc("/detail/", s.DetailHandler)
	mux.HandleFunc("/download", s.DownloadHandler)
	mux.HandleFunc("/", s.HomeHandler)  // Must be last

	// API routes
	mux.HandleFunc("/api/stats", s.APIStatsHandler)
	mux.HandleFunc("/api/properties", s.APIPropertiesHandler)
	mux.HandleFunc("/api/health", s.HealthHandler)
}
