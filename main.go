package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

// Property struct
type Property struct {
	ID          string        `json:"_id"`
	Title       string        `json:"title"`
	Slug        Slug          `json:"slug"`
	Developer   Reference     `json:"developer"`
	Description string        `json:"description"`
	MapURL      string        `json:"mapUrl"`
	GeoLocation GeoLocation   `json:"geoLocation"`
	MinPrice    float32       `json:"minPrice"`
	MaxPrice    float32       `json:"maxPrice"`
	Facilities  []Facility    `json:"facilities"`
	Photos      []SanityImage `json:"photos"`
	Built       int           `json:"built"`
	CreatedAt   string        `json:"createdAt"`
}

type Slug struct {
	Current string `json:"current"`
	Type    string `json:"_type"`
}

type Reference struct {
	Ref  string `json:"_ref"`
	Type string `json:"_type"`
}

type GeoLocation struct {
	Type string  `json:"_type"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

type Facility struct {
	FacilityType Reference     `json:"facilityType"`
	FacilityName string        `json:"facilityName"`
	Description  string        `json:"description"`
	Photos       []SanityImage `json:"photos"`
}

type SanityImage struct {
	Key   string `json:"_key"`
	Type  string `json:"_type"`
	Asset Asset  `json:"asset"`
}

type Asset struct {
	Ref  string `json:"_ref"`
	Type string `json:"_type"`
}

// Mock data for now (later, you'll fetch this from Sanity)
var properties []Property

// const sanityAPI = "https://tq4u5fnu.api.sanity.io/v1/data/query/production?query="

func main() {

	// uncomment here for localhost testing
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file:", err)
    }

	// Fetch SANITY_API_URL from environment variables
	sanityAPI := os.Getenv("SANITY_API_URL")
	if sanityAPI == "" {
		log.Fatal("SANITY_API_URL is not set in the environment")
	}

	// Initialize mux router
	r := mux.NewRouter()

	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}), // Allow requests from all origins
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "X-API-Key"}),
	)

	// Create a new handler with CORS middleware
	handler := cors(r)


	// Route handles & endpoints
	r.HandleFunc("/properties", GetProperties).Methods("GET")
	r.HandleFunc("/properties/{slug}", GetPropertyBySlug).Methods("GET")

	// Start fetching properties from Sanity every hour
	go func() {
		for {
			log.Println("Starting property fetch from Sanity...")
			start := time.Now()

			fetchPropertiesFromSanity(sanityAPI)

			log.Printf("Property fetch completed in %s", time.Since(start))
			log.Println("Next update will occur in 1 hour.")

			time.Sleep(1 * time.Hour)
		}
	}()

	// Start server
	// log.Println("Server is running on :8000")
	// log.Fatal(http.ListenAndServe(":8000", handler))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Default port to 8000 if PORT environment variable is not set
	}
	fmt.Println("Server is running on port:", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// GetProperties returns all properties
func GetProperties(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(properties)
}

// fetchPropertiesFromSanity fetches properties from Sanity and updates the in-memory `properties` slice
func fetchPropertiesFromSanity(sanityAPI string) {
	query := "*[_type == \"property\"]"
	encodedQuery := url.QueryEscape(query)
	url := sanityAPI + encodedQuery

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Failed to fetch properties from Sanity:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Sanity API returned non-200 status:", resp.Status)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read Sanity API response:", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Println("Failed to parse JSON from Sanity API:", err)
		return
	}

	if propertiesData, ok := result["result"].([]interface{}); ok {
		newProperties := []Property{}
		for _, propertyData := range propertiesData {
			propertyBytes, _ := json.Marshal(propertyData)
			var property Property
			if err := json.Unmarshal(propertyBytes, &property); err != nil {
				log.Println("Failed to unmarshal property:", err)
				continue
			}
			newProperties = append(newProperties, property)
		}
		properties = newProperties
		log.Println("Properties successfully updated from Sanity.")
	} else {
		log.Println("No properties found in Sanity API response.")
	}
}

// GetPropertyBySlug returns a single property by slug
func GetPropertyBySlug(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for _, item := range properties {
		if item.Slug.Current == params["slug"] {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(item)
			return
		}
	}
	http.Error(w, "Property not found", http.StatusNotFound)
}
