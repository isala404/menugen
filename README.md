# Menu Visualizer

A proof-of-concept application that digitizes restaurant menus by processing uploaded photos using AI to extract menu items and generate illustrative images for each dish.

## Architecture

The application consists of:
- **React Frontend**: Single-page application for uploading menu photos and displaying digital menus
- **Go Backend**: REST API that processes images using OpenAI GPT-4o and Replicate API
- **PostgreSQL Database**: Stores extracted menu data and generated images

## Features

- 📸 Upload menu photos via drag-and-drop or file selection
- 🤖 AI-powered text extraction using OpenAI GPT-4o vision
- 🎨 Automatic dish image generation using Replicate's FLUX.1 model
- 💾 Persistent storage of menus and generated images
- 🔐 Choreo managed authentication support
- 📱 Responsive design for mobile and desktop

## External API Integrations

### OpenAI GPT-4o
- **Purpose**: Extract structured menu data from uploaded images
- **Integration**: Uses vision capabilities to read menu text and return XML-structured data
- **Configuration**: Via Choreo Marketplace connection

### Replicate FLUX.1
- **Purpose**: Generate appealing food images for each extracted dish
- **Integration**: Creates photorealistic images based on dish names and descriptions
- **Configuration**: Via Choreo Marketplace connection

## Development Setup

### Prerequisites
- Node.js 18+
- Go 1.21+
- PostgreSQL
- OpenAI API key
- Replicate API key

### Local Development

1. **Backend Setup**:
   ```bash
   cd backend
   go mod tidy
   
   # Set environment variables
   export DATABASE_URL="host=localhost user=postgres password=postgres dbname=menugen port=5432 sslmode=disable"
   export OPENAI_API_KEY="your-openai-key"
   export REPLICATE_API_KEY="your-replicate-key"
   
   go run main.go
   ```

2. **Frontend Setup**:
   ```bash
   cd frontend
   npm install
   npm start
   ```

3. **Database Setup**:
   ```sql
   CREATE DATABASE menugen;
   ```

## Choreo Deployment

This application is designed to be deployed on WSO2 Choreo with the following components:

### Service Component (Backend)
- **Type**: Go service
- **Port**: 8080
- **Endpoints**: 
  - `POST /api/menu` - Upload and process menu
  - `GET /api/menu/{id}` - Retrieve processed menu
  - `GET /health` - Health check

### Web Application Component (Frontend)
- **Type**: React web app
- **Build**: Creates optimized production build
- **Authentication**: Choreo managed auth enabled

### External Connections
- **OpenAI**: Connected via Choreo Marketplace
- **Replicate**: Connected via Choreo Marketplace

### Database
- **Type**: PostgreSQL managed database
- **Connection**: Via Choreo database connection

## API Endpoints

### POST /api/menu
Upload and process a menu image.

**Request**:
- Method: POST
- Content-Type: multipart/form-data
- Body: image file

**Response**:
```json
{
  "success": true,
  "menu": {
    "id": 1,
    "restaurant_name": "Example Restaurant",
    "dishes": [
      {
        "id": 1,
        "name": "Margherita Pizza",
        "description": "Fresh tomatoes, mozzarella, basil",
        "price": "$12.99",
        "image_url": "https://generated-image-url.jpg"
      }
    ]
  }
}
```

### GET /api/menu/{id}
Retrieve a processed menu by ID.

**Response**:
```json
{
  "id": 1,
  "restaurant_name": "Example Restaurant",
  "dishes": [...],
  "created_at": "2025-08-20T12:00:00Z"
}
```

## Environment Variables

### Backend
- `DATABASE_URL`: PostgreSQL connection string
- `OPENAI_API_KEY`: OpenAI API key (injected by Choreo)
- `OPENAI_SERVICE_URL`: OpenAI service URL (injected by Choreo)
- `REPLICATE_API_KEY`: Replicate API key (injected by Choreo)
- `REPLICATE_SERVICE_URL`: Replicate service URL (injected by Choreo)
- `PORT`: Server port (default: 8080)

### Frontend
- `REACT_APP_API_URL`: Backend API URL (configured by Choreo)

## Data Flow

1. User uploads menu image via React frontend
2. Frontend sends image to Go backend via REST API
3. Backend processes image with OpenAI GPT-4o to extract menu structure
4. For each extracted dish, backend generates image using Replicate API
5. Menu data and images are stored in PostgreSQL database
6. Frontend receives processed menu and displays interactive digital menu

## Security

- **Authentication**: Choreo managed authentication for web application
- **CORS**: Configured for cross-origin requests
- **Input Validation**: File type and size validation
- **Error Handling**: Comprehensive error handling and logging

## Future Enhancements

- **Real-time Updates**: WebSocket integration for live processing updates
- **Image Optimization**: Compress and optimize generated images
- **Menu Categories**: Support for menu sections and categories
- **Multi-language**: Support for multiple languages
- **Advanced Customization**: Theming and styling options for digital menus

## License

This project is a proof-of-concept and is provided as-is for demonstration purposes.
