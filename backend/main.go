package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database Models
type Menu struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"created_at"`
	RestaurantName string    `json:"restaurant_name"`
	Dishes         []Dish    `json:"dishes" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Dish struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	MenuID      uint      `json:"menu_id"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	ImageBlob   []byte    `json:"-" gorm:"type:bytea"`
	ImageURL    string    `json:"image_url" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

// XML structures for OpenAI response parsing
type MenuXML struct {
	XMLName        xml.Name  `xml:"menu"`
	RestaurantName string    `xml:"restaurant_name"`
	Dishes         []DishXML `xml:"dishes>dish"`
}

type DishXML struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Price       string `xml:"price"`
}

// API Request/Response structures
type OpenAIRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// Replicate API structures
type ReplicateRequest struct {
	Input ReplicateInput `json:"input"`
}

type ReplicateInput struct {
	Prompt string `json:"prompt"`
}

type ReplicateResponse struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Output interface{} `json:"output"`
	URLs   struct {
		Get string `json:"get"`
	} `json:"urls"`
}

// Global variables
var db *gorm.DB
var openaiAPIKey string
var openaiServiceURL string
var replicateAPIKey string
var replicateServiceURL string

func init() {
	// Get environment variables - these will be injected by Choreo connections
	openaiAPIKey = os.Getenv("OPENAI_API_KEY")
	openaiServiceURL = os.Getenv("OPENAI_SERVICE_URL")
	if openaiServiceURL == "" {
		openaiServiceURL = "https://api.openai.com/v1"
	}

	replicateAPIKey = os.Getenv("REPLICATE_API_KEY")
	replicateServiceURL = os.Getenv("REPLICATE_SERVICE_URL")
	if replicateServiceURL == "" {
		replicateServiceURL = "https://api.replicate.com/v1"
	}
}

func main() {
	// Initialize database
	var err error
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback for local development
		dsn = "host=localhost user=postgres password=postgres dbname=menugen port=5432 sslmode=disable"
	}

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&Menu{}, &Dish{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Setup routes
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/api/menu", handleMenuUpload).Methods("POST")
	router.HandleFunc("/api/menu/{id}", handleGetMenu).Methods("GET")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // In production, specify your frontend domain
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func handleMenuUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to get image file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	imageBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// Extract menu data using OpenAI
	menuData, err := extractMenuWithOpenAI(imageBytes)
	if err != nil {
		log.Printf("OpenAI extraction error: %v", err)
		http.Error(w, "Failed to extract menu data", http.StatusInternalServerError)
		return
	}

	// Create menu in database
	menu := Menu{
		RestaurantName: menuData.RestaurantName,
		CreatedAt:      time.Now(),
	}

	if err := db.Create(&menu).Error; err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to save menu", http.StatusInternalServerError)
		return
	}

	// Process each dish and generate images
	var dishes []Dish
	for _, dishXML := range menuData.Dishes {
		imageURL, err := generateDishImage(dishXML.Name, dishXML.Description)
		if err != nil {
			log.Printf("Image generation error for dish %s: %v", dishXML.Name, err)
			// Continue with other dishes even if one fails
		}

		dish := Dish{
			MenuID:      menu.ID,
			Name:        dishXML.Name,
			Description: dishXML.Description,
			Price:       dishXML.Price,
			ImageURL:    imageURL,
			CreatedAt:   time.Now(),
		}

		if err := db.Create(&dish).Error; err != nil {
			log.Printf("Failed to save dish %s: %v", dishXML.Name, err)
			continue
		}

		dishes = append(dishes, dish)
	}

	// Update menu with dishes
	menu.Dishes = dishes

	response := map[string]interface{}{
		"success": true,
		"menu":    menu,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetMenu(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "Invalid menu ID", http.StatusBadRequest)
		return
	}

	var menu Menu
	if err := db.Preload("Dishes").First(&menu, id).Error; err != nil {
		http.Error(w, "Menu not found", http.StatusNotFound)
		return
	}

	// Convert image blobs to data URLs for display
	for i := range menu.Dishes {
		if len(menu.Dishes[i].ImageBlob) > 0 {
			encoded := base64.StdEncoding.EncodeToString(menu.Dishes[i].ImageBlob)
			menu.Dishes[i].ImageURL = "data:image/webp;base64," + encoded
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(menu)
}

func extractMenuWithOpenAI(imageBytes []byte) (*MenuXML, error) {
	// Encode image to base64
	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	dataURL := "data:image/jpeg;base64," + encoded

	// Prepare OpenAI request
	request := OpenAIRequest{
		Model:     "gpt-4o",
		MaxTokens: 2000,
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: `Please analyze this menu image and extract all the dish information. Return the data in the following XML format:

<menu>
  <restaurant_name>Name of Restaurant (if visible)</restaurant_name>
  <dishes>
    <dish>
      <name>Dish Name</name>
      <description>Brief description</description>
      <price>$X.XX</price>
    </dish>
    <!-- Repeat for each dish -->
  </dishes>
</menu>

Extract all visible dishes with their names, descriptions (if available), and prices. If no restaurant name is visible, leave it empty. Be thorough and capture all menu items you can see.`,
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: dataURL,
						},
					},
				},
			},
		},
	}

	// Make request to OpenAI
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", openaiServiceURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
	}

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	// Parse XML from response
	content := openaiResp.Choices[0].Message.Content

	// Extract XML content between ```xml and ``` if present
	if strings.Contains(content, "```xml") {
		start := strings.Index(content, "```xml") + 6
		end := strings.LastIndex(content, "```")
		if end > start {
			content = content[start:end]
		}
	} else if strings.Contains(content, "<menu>") {
		// Extract just the XML portion
		start := strings.Index(content, "<menu>")
		end := strings.LastIndex(content, "</menu>") + 7
		if end > start {
			content = content[start:end]
		}
	}

	var menuData MenuXML
	if err := xml.Unmarshal([]byte(content), &menuData); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	return &menuData, nil
}

func generateDishImage(dishName, description string) (string, error) {
	// Create a descriptive prompt for image generation
	prompt := fmt.Sprintf("Professional food photography of %s", dishName)
	if description != "" {
		prompt += fmt.Sprintf(", %s", description)
	}
	prompt += ", appetizing, restaurant quality, well lit, beautiful presentation"

	// Prepare Replicate request
	request := ReplicateRequest{
		Input: ReplicateInput{
			Prompt: prompt,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make request to Replicate
	req, err := http.NewRequest("POST", replicateServiceURL+"/models/black-forest-labs/flux-dev/predictions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+replicateAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Replicate API error %d: %s", resp.StatusCode, string(body))
	}

	var replicateResp ReplicateResponse
	if err := json.NewDecoder(resp.Body).Decode(&replicateResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	// Poll for completion
	for i := 0; i < 60; i++ { // Poll for up to 60 seconds
		time.Sleep(1 * time.Second)

		// Check status
		statusReq, err := http.NewRequest("GET", replicateResp.URLs.Get, nil)
		if err != nil {
			continue
		}
		statusReq.Header.Set("Authorization", "Bearer "+replicateAPIKey)

		statusResp, err := client.Do(statusReq)
		if err != nil {
			continue
		}

		var statusData ReplicateResponse
		json.NewDecoder(statusResp.Body).Decode(&statusData)
		statusResp.Body.Close()

		if statusData.Status == "succeeded" && statusData.Output != nil {
			// Extract image URL from output
			if outputs, ok := statusData.Output.([]interface{}); ok && len(outputs) > 0 {
				if imageURL, ok := outputs[0].(string); ok {
					return imageURL, nil
				}
			}
		} else if statusData.Status == "failed" {
			return "", fmt.Errorf("image generation failed")
		}
	}

	return "", fmt.Errorf("image generation timed out")
}
