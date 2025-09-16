package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database Models
type Menu struct {
	ID              string        `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	OriginalFile    string        `json:"original_filename"`
	ImageHash       string        `json:"image_hash" gorm:"uniqueIndex"`
	Status          string        `json:"status" gorm:"type:varchar(20);default:'PENDING'"`
	FailureReason   *string       `json:"failure_reason"`
	TotalDishes     int           `json:"total_dishes"`
	ProcessedDishes int           `json:"processed_dishes"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	CompletedAt     *time.Time    `json:"completed_at"`
	Sections        []MenuSection `json:"sections,omitempty" gorm:"foreignKey:MenuID"`
	Dishes          []Dish        `json:"dishes,omitempty" gorm:"foreignKey:MenuID"`
}

type MenuSection struct {
	ID       string `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MenuID   string `json:"menu_id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type Dish struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MenuID         string    `json:"menu_id"`
	SectionID      *string   `json:"section_id"`
	Name           string    `json:"name"`
	PriceCents     *int      `json:"price_cents"`
	Currency       string    `json:"currency" gorm:"default:'USD'"`
	RawPriceString *string   `json:"raw_price_string"`
	Description    *string   `json:"description"`
	ImageURL       *string   `json:"image_url"`
	Status         string    `json:"status" gorm:"type:varchar(20);default:'PENDING'"`
	FailureReason  *string   `json:"failure_reason"`
	Position       int       `json:"position"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Request/Response Models
type MenuUploadResponse struct {
	MenuID string `json:"menu_id"`
	Status string `json:"status"`
}

type MenuStatusResponse struct {
	MenuID   string                 `json:"menu_id"`
	Status   string                 `json:"status"`
	Progress *MenuProgress          `json:"progress,omitempty"`
	Menu     *MenuStructureResponse `json:"menu,omitempty"`
	Error    *ErrorResponse         `json:"error,omitempty"`
}

type MenuProgress struct {
	ProcessedDishes int `json:"processed_dishes"`
	TotalDishes     int `json:"total_dishes"`
}

type MenuStructureResponse struct {
	ID       string                `json:"id"`
	Status   string                `json:"status"`
	Sections []MenuSectionResponse `json:"sections"`
	Dishes   []DishResponse        `json:"dishes"`
}

type MenuSectionResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type DishResponse struct {
	ID             string  `json:"id"`
	SectionID      *string `json:"section_id"`
	Name           string  `json:"name"`
	PriceCents     *int    `json:"price_cents"`
	Currency       string  `json:"currency"`
	RawPriceString *string `json:"raw_price_string"`
	Description    *string `json:"description"`
	ImageURL       *string `json:"image_url"`
	Status         string  `json:"status"`
	Position       int     `json:"position"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OpenAI Types
type OpenAIVisionRequest struct {
	Model          string                `json:"model"`
	Messages       []OpenAIMessage       `json:"messages"`
	ResponseFormat *OpenAIResponseFormat `json:"response_format,omitempty"`
	MaxTokens      int                   `json:"max_tokens"`
}

type OpenAIMessage struct {
	Role    string          `json:"role"`
	Content []OpenAIContent `json:"content"`
}

type OpenAIContent struct {
	Type     string          `json:"type"`
	Text     *string         `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

type OpenAIImageURL struct {
	URL string `json:"url"`
}

type OpenAIResponseFormat struct {
	Type       string           `json:"type"`
	JSONSchema OpenAIJSONSchema `json:"json_schema"`
}

type OpenAIJSONSchema struct {
	Name   string      `json:"name"`
	Strict bool        `json:"strict"`
	Schema interface{} `json:"schema"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
}

type OpenAIChoice struct {
	Message OpenAIResponseMessage `json:"message"`
}

type OpenAIResponseMessage struct {
	Content string `json:"content"`
}

type OpenAITextRequest struct {
	Model          string                `json:"model"`
	Messages       []OpenAITextMessage   `json:"messages"`
	ResponseFormat *OpenAIResponseFormat `json:"response_format,omitempty"`
	MaxTokens      int                   `json:"max_tokens"`
}

type OpenAITextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

type OpenAIImageResponse struct {
	Data []OpenAIImageData `json:"data"`
}

type OpenAIImageData struct {
	URL string `json:"url"`
}

// Replicate Types
type ReplicateRequest struct {
	Input ReplicateInput `json:"input"`
}

type ReplicateInput struct {
	Prompt            string  `json:"prompt"`
	AspectRatio       string  `json:"aspect_ratio"`
	NumOutputs        int     `json:"num_outputs"`
	NumInferenceSteps int     `json:"num_inference_steps"`
	Guidance          float64 `json:"guidance"`
	OutputFormat      string  `json:"output_format"`
	OutputQuality     int     `json:"output_quality"`
	GoFast            bool    `json:"go_fast"`
}

type ReplicateResponse struct {
	ID     string        `json:"id"`
	Status string        `json:"status"`
	Output []string      `json:"output"`
	URLs   ReplicateURLs `json:"urls"`
}

type ReplicateURLs struct {
	Get string `json:"get"`
}

// Structured Menu Schema for OpenAI
type StructuredMenu struct {
	Sections []StructuredSection `json:"sections"`
}

type StructuredSection struct {
	Name   string           `json:"name"`
	Dishes []StructuredDish `json:"dishes"`
}

type StructuredDish struct {
	Name  string  `json:"name"`
	Price *string `json:"price"`
}

// Global variables
var (
	db     *gorm.DB
	zapLog *zap.Logger
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize logger
	var err error
	zapLog, err = zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer zapLog.Sync()

	// Initialize database
	if err := initDB(); err != nil {
		zapLog.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Routes
	api := r.Group("/api")
	{
		api.POST("/menu", uploadMenuHandler)
		api.GET("/menu/:id", getMenuHandler)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	zapLog.Info("Starting server", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		zapLog.Fatal("Failed to start server", zap.Error(err))
	}
}

func initDB() error {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSL_MODE")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbName == "" {
		dbName = "menugen"
	}
	if dbSSLMode == "" {
		dbSSLMode = "require"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&Menu{}, &MenuSection{}, &Dish{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	zapLog.Info("Database initialized successfully")
	return nil
}

func uploadMenuHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrorResponse{
				Code:    "MISSING_FILE",
				Message: "No image file provided",
			},
		})
		return
	}
	defer file.Close()

	// Validate file size (8MB limit)
	if header.Size > 8*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrorResponse{
				Code:    "FILE_TOO_LARGE",
				Message: "File size exceeds 8MB limit",
			},
		})
		return
	}

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrorResponse{
				Code:    "INVALID_FILE_TYPE",
				Message: "File must be an image",
			},
		})
		return
	}

	// Read file content and calculate hash
	fileContent, err := io.ReadAll(file)
	if err != nil {
		zapLog.Error("Failed to read file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to process file",
			},
		})
		return
	}

	hash := sha256.Sum256(fileContent)
	imageHash := fmt.Sprintf("%x", hash)

	// Check if menu with same hash already exists
	var existingMenu Menu
	if err := db.Where("image_hash = ?", imageHash).First(&existingMenu).Error; err == nil {
		c.JSON(http.StatusOK, MenuUploadResponse{
			MenuID: existingMenu.ID,
			Status: existingMenu.Status,
		})
		return
	}

	// Create new menu record
	menu := Menu{
		ID:              uuid.New().String(),
		OriginalFile:    header.Filename,
		ImageHash:       imageHash,
		Status:          "PENDING",
		TotalDishes:     0,
		ProcessedDishes: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := db.Create(&menu).Error; err != nil {
		zapLog.Error("Failed to create menu", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": ErrorResponse{
				Code:    "DATABASE_ERROR",
				Message: "Failed to create menu",
			},
		})
		return
	}

	// Start async processing
	go processMenu(menu.ID, fileContent)

	c.JSON(http.StatusAccepted, MenuUploadResponse{
		MenuID: menu.ID,
		Status: menu.Status,
	})
}

func getMenuHandler(c *gin.Context) {
	menuID := c.Param("id")

	var menu Menu
	if err := db.Preload("Sections").Preload("Dishes").Where("id = ?", menuID).First(&menu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": ErrorResponse{
				Code:    "MENU_NOT_FOUND",
				Message: "Menu not found",
			},
		})
		return
	}

	response := MenuStatusResponse{
		MenuID: menu.ID,
		Status: menu.Status,
	}

	if menu.Status == "PROCESSING" || menu.Status == "COMPLETE" {
		response.Progress = &MenuProgress{
			ProcessedDishes: menu.ProcessedDishes,
			TotalDishes:     menu.TotalDishes,
		}
	}

	if menu.Status == "COMPLETE" {
		sections := make([]MenuSectionResponse, len(menu.Sections))
		for i, section := range menu.Sections {
			sections[i] = MenuSectionResponse{
				ID:       section.ID,
				Name:     section.Name,
				Position: section.Position,
			}
		}

		dishes := make([]DishResponse, len(menu.Dishes))
		for i, dish := range menu.Dishes {
			dishes[i] = DishResponse{
				ID:             dish.ID,
				SectionID:      dish.SectionID,
				Name:           dish.Name,
				PriceCents:     dish.PriceCents,
				Currency:       dish.Currency,
				RawPriceString: dish.RawPriceString,
				Description:    dish.Description,
				ImageURL:       dish.ImageURL,
				Status:         dish.Status,
				Position:       dish.Position,
			}
		}

		response.Menu = &MenuStructureResponse{
			ID:       menu.ID,
			Status:   menu.Status,
			Sections: sections,
			Dishes:   dishes,
		}
	}

	if menu.Status == "FAILED" && menu.FailureReason != nil {
		response.Error = &ErrorResponse{
			Code:    "PROCESSING_FAILED",
			Message: *menu.FailureReason,
		}
	}

	c.JSON(http.StatusOK, response)
}

func processMenu(menuID string, imageContent []byte) {
	zapLog.Info("Starting menu processing", zap.String("menuID", menuID))

	// Update status to PROCESSING
	if err := db.Model(&Menu{}).Where("id = ?", menuID).Updates(map[string]interface{}{
		"status":     "PROCESSING",
		"updated_at": time.Now(),
	}).Error; err != nil {
		zapLog.Error("Failed to update menu status", zap.String("menuID", menuID), zap.Error(err))
		return
	}

	// Step 1: OCR + Structure using OpenAI Vision
	structuredMenu, err := extractMenuStructure(imageContent)
	if err != nil {
		failMenu(menuID, "Failed to extract menu structure: "+err.Error())
		return
	}

	// Step 2: Create menu sections and dishes
	var totalDishes int
	var dishIDs []string

	tx := db.Begin()

	for sectionIdx, section := range structuredMenu.Sections {
		menuSection := MenuSection{
			ID:       uuid.New().String(),
			MenuID:   menuID,
			Name:     section.Name,
			Position: sectionIdx,
		}

		if err := tx.Create(&menuSection).Error; err != nil {
			tx.Rollback()
			failMenu(menuID, "Failed to create menu section: "+err.Error())
			return
		}

		for dishIdx, dish := range section.Dishes {
			var priceCents *int
			if dish.Price != nil && *dish.Price != "" {
				if cents := extractPriceCents(*dish.Price); cents > 0 {
					priceCents = &cents
				}
			}

			dishRecord := Dish{
				ID:             uuid.New().String(),
				MenuID:         menuID,
				SectionID:      &menuSection.ID,
				Name:           dish.Name,
				PriceCents:     priceCents,
				Currency:       "USD",
				RawPriceString: dish.Price,
				Status:         "PENDING",
				Position:       dishIdx,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			if err := tx.Create(&dishRecord).Error; err != nil {
				tx.Rollback()
				failMenu(menuID, "Failed to create dish: "+err.Error())
				return
			}

			dishIDs = append(dishIDs, dishRecord.ID)
			totalDishes++
		}
	}

	// Update menu with total dishes count
	if err := tx.Model(&Menu{}).Where("id = ?", menuID).Updates(map[string]interface{}{
		"total_dishes": totalDishes,
		"updated_at":   time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		failMenu(menuID, "Failed to update menu: "+err.Error())
		return
	}

	tx.Commit()

	// Step 3: Enhance each dish with description and image
	var processedCount int
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Limit concurrent processing

	for _, dishID := range dishIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if enhanceDish(id) {
				processedCount++
			}

			// Update progress
			db.Model(&Menu{}).Where("id = ?", menuID).Updates(map[string]interface{}{
				"processed_dishes": processedCount,
				"updated_at":       time.Now(),
			})
		}(dishID)
	}

	wg.Wait()

	// Complete the menu
	completedAt := time.Now()
	if err := db.Model(&Menu{}).Where("id = ?", menuID).Updates(map[string]interface{}{
		"status":       "COMPLETE",
		"updated_at":   completedAt,
		"completed_at": &completedAt,
	}).Error; err != nil {
		zapLog.Error("Failed to complete menu", zap.String("menuID", menuID), zap.Error(err))
		return
	}

	zapLog.Info("Menu processing completed", zap.String("menuID", menuID), zap.Int("totalDishes", totalDishes))
}

func extractMenuStructure(imageContent []byte) (*StructuredMenu, error) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	// Convert image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageContent)
	imageURL := "data:image/jpeg;base64," + base64Image

	// Define the schema for structured response
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sections": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
						},
						"dishes": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name": map[string]interface{}{
										"type": "string",
									},
									"price": map[string]interface{}{
										"type": "string",
									},
								},
								"required": []string{"name"},
							},
						},
					},
					"required": []string{"name", "dishes"},
				},
			},
		},
		"required": []string{"sections"},
	}

	request := OpenAIVisionRequest{
		Model: "gpt-4o",
		Messages: []OpenAIMessage{
			{
				Role: "user",
				Content: []OpenAIContent{
					{
						Type: "text",
						Text: stringPtr("Extract the menu structure from this image. Organize dishes into sections. Include dish names and prices if visible. Return the data as structured JSON."),
					},
					{
						Type: "image_url",
						ImageURL: &OpenAIImageURL{
							URL: imageURL,
						},
					},
				},
			},
		},
		ResponseFormat: &OpenAIResponseFormat{
			Type: "json_schema",
			JSONSchema: OpenAIJSONSchema{
				Name:   "menu_structure",
				Strict: false,
				Schema: schema,
			},
		},
		MaxTokens: 2000,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	var structuredMenu StructuredMenu
	if err := json.Unmarshal([]byte(openaiResp.Choices[0].Message.Content), &structuredMenu); err != nil {
		return nil, fmt.Errorf("failed to unmarshal structured menu: %w", err)
	}

	return &structuredMenu, nil
}

func enhanceDish(dishID string) bool {
	var dish Dish
	if err := db.Where("id = ?", dishID).First(&dish).Error; err != nil {
		zapLog.Error("Failed to find dish", zap.String("dishID", dishID), zap.Error(err))
		return false
	}

	// Generate description
	description, err := generateDishDescription(dish.Name)
	if err != nil {
		zapLog.Error("Failed to generate description", zap.String("dishID", dishID), zap.Error(err))
		markDishFailed(dishID, "Failed to generate description: "+err.Error())
		return false
	}

	// Generate image
	imageURL, err := generateDishImage(dish.Name)
	if err != nil {
		zapLog.Error("Failed to generate image", zap.String("dishID", dishID), zap.Error(err))
		// Continue with description but no image
	}

	// Update dish
	updates := map[string]interface{}{
		"description": description,
		"status":      "COMPLETE",
		"updated_at":  time.Now(),
	}

	if imageURL != nil {
		updates["image_url"] = *imageURL
	}

	if err := db.Model(&Dish{}).Where("id = ?", dishID).Updates(updates).Error; err != nil {
		zapLog.Error("Failed to update dish", zap.String("dishID", dishID), zap.Error(err))
		return false
	}

	return true
}

func generateDishDescription(dishName string) (string, error) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	request := OpenAITextRequest{
		Model: "gpt-4o-mini",
		Messages: []OpenAITextMessage{
			{
				Role:    "system",
				Content: "You are a food writer. Generate a brief, appetizing description (1-2 sentences) for the given dish name. Be descriptive but concise.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Generate a description for this dish: %s", dishName),
			},
		},
		MaxTokens: 100,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	return strings.TrimSpace(openaiResp.Choices[0].Message.Content), nil
}

func generateDishImage(dishName string) (*string, error) {
	replicateAPIKey := os.Getenv("REPLICATE_API_KEY")
	if replicateAPIKey == "" {
		return nil, fmt.Errorf("REPLICATE_API_KEY not set")
	}

	prompt := fmt.Sprintf("A beautiful, appetizing photo of %s, food photography, professional lighting, clean background", dishName)

	request := ReplicateRequest{
		Input: ReplicateInput{
			Prompt:            prompt,
			AspectRatio:       "1:1",
			NumOutputs:        1,
			NumInferenceSteps: 28,
			Guidance:          3.5,
			OutputFormat:      "webp",
			OutputQuality:     80,
			GoFast:            true,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.replicate.com/v1/models/black-forest-labs/flux-dev/predictions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+replicateAPIKey)
	req.Header.Set("Prefer", "wait")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Replicate API error: %s", string(body))
	}

	var replicateResp ReplicateResponse
	if err := json.NewDecoder(resp.Body).Decode(&replicateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// If image is ready immediately
	if len(replicateResp.Output) > 0 {
		return &replicateResp.Output[0], nil
	}

	// Poll for completion if not ready
	if replicateResp.URLs.Get != "" {
		return pollReplicateResult(replicateResp.URLs.Get, replicateAPIKey)
	}

	return nil, fmt.Errorf("no output or polling URL available")
}

func pollReplicateResult(pollURL, apiKey string) (*string, error) {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(time.Duration(i+1) * time.Second)

		req, err := http.NewRequest("GET", pollURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create polling request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		var result ReplicateResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if result.Status == "succeeded" && len(result.Output) > 0 {
			return &result.Output[0], nil
		}

		if result.Status == "failed" {
			return nil, fmt.Errorf("image generation failed")
		}
	}

	return nil, fmt.Errorf("polling timeout")
}

func extractPriceCents(priceStr string) int {
	// Simple price extraction - look for numbers
	cleaned := strings.ReplaceAll(priceStr, "$", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)

	if price, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return int(price * 100) // Convert to cents
	}

	return 0
}

func failMenu(menuID, reason string) {
	if err := db.Model(&Menu{}).Where("id = ?", menuID).Updates(map[string]interface{}{
		"status":         "FAILED",
		"failure_reason": reason,
		"updated_at":     time.Now(),
	}).Error; err != nil {
		zapLog.Error("Failed to update menu failure", zap.String("menuID", menuID), zap.Error(err))
	}
}

func markDishFailed(dishID, reason string) {
	if err := db.Model(&Dish{}).Where("id = ?", dishID).Updates(map[string]interface{}{
		"status":         "FAILED",
		"failure_reason": reason,
		"updated_at":     time.Now(),
	}).Error; err != nil {
		zapLog.Error("Failed to update dish failure", zap.String("dishID", dishID), zap.Error(err))
	}
}

func stringPtr(s string) *string {
	return &s
}
