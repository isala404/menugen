# MenuGen - AI-Powered Menu Digitization

MenuGen is a full-stack application that transforms photographed restaurant menus into structured, queryable digital menus using AI. It extracts dish information, generates descriptions, and creates dish images using OpenAI's vision models and Replicate's image generation.

## Features

- **Image Upload**: Upload menu photos (JPEG, PNG, WEBP up to 8MB)
- **AI Menu Extraction**: Extract menu structure using OpenAI's GPT-4 Vision
- **Dish Enhancement**: Generate AI descriptions and dish images
- **Real-time Progress**: Poll-based status updates during processing
- **Structured Output**: Get organized menu data with sections and dishes
- **Responsive UI**: Modern React frontend with Tailwind CSS

## Architecture

### Backend (Go)
- **Framework**: Gin HTTP router
- **Database**: PostgreSQL with GORM
- **Logging**: Zap structured logging
- **AI Services**: OpenAI API for vision/text, Replicate API for images
- **Processing**: Async menu processing with progress tracking

### Frontend (React)
- **Framework**: React with Vite
- **Styling**: Tailwind CSS (via CDN)
- **State Management**: React hooks
- **Communication**: Fetch API with polling

## Prerequisites

- Go 1.19+
- Node.js 18+
- PostgreSQL 12+
- OpenAI API key
- Replicate API key

## Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd menugen
```

### 2. Backend Setup

```bash
cd backend

# Install dependencies
go mod download

# Copy environment file
cp .env.example .env

# Edit .env with your configuration
# - Database connection details
# - OpenAI API key
# - Replicate API key
```

#### Environment Variables

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_postgres_password
DB_NAME=menugen
DB_SSL_MODE=disable

# API Keys
OPENAI_API_KEY=your_openai_api_key_here
REPLICATE_API_KEY=your_replicate_api_key_here

# Server Configuration
PORT=8080
```

### 3. Database Setup

Create a PostgreSQL database:

```sql
CREATE DATABASE menugen;
```

The application will automatically create the required tables on startup using GORM's auto-migration.

### 4. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Copy environment file (optional)
cp .env.example .env

# Edit .env if needed (defaults to localhost:8080)
```

## Running the Application

### Start the Database

```bash
# Using Docker (recommended)
docker run --name menugen-postgres \
  -e POSTGRES_DB=menugen \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=your_password \
  -p 5432:5432 \
  -d postgres:15
```

### Start the Backend

```bash
cd backend
go run .
```

The backend will start on `http://localhost:8080`

### Start the Frontend

```bash
cd frontend
npm run dev
```

The frontend will start on `http://localhost:5173`

## API Endpoints

### POST /api/menu
Upload a menu image for processing.

**Request:**
- Method: `POST`
- Content-Type: `multipart/form-data`
- Body: `image` file field

**Response:**
```json
{
  "menu_id": "uuid",
  "status": "PENDING"
}
```

### GET /api/menu/:id
Get menu processing status and results.

**Response:**
```json
{
  "menu_id": "uuid",
  "status": "COMPLETE",
  "progress": {
    "processed_dishes": 10,
    "total_dishes": 10
  },
  "menu": {
    "id": "uuid",
    "status": "COMPLETE",
    "sections": [
      {
        "id": "uuid",
        "name": "Appetizers",
        "position": 0
      }
    ],
    "dishes": [
      {
        "id": "uuid",
        "section_id": "uuid",
        "name": "Caesar Salad",
        "price_cents": 1200,
        "currency": "USD",
        "description": "Fresh romaine lettuce with...",
        "image_url": "https://...",
        "status": "COMPLETE",
        "position": 0
      }
    ]
  }
}
```

## Database Schema

### Tables

- **menus**: Main menu records with processing status
- **menu_sections**: Menu sections (Appetizers, Mains, etc.)
- **dishes**: Individual dish records with enhanced data

### Status Flow

1. `PENDING` - Menu uploaded, queued for processing
2. `PROCESSING` - AI extraction and enhancement in progress
3. `COMPLETE` - All dishes processed successfully
4. `FAILED` - Processing failed with error reason

## Third-Party Integrations

### OpenAI Integration
- **Vision API**: Extract menu structure from images
- **Chat API**: Generate dish descriptions
- Uses structured JSON responses for reliable parsing

### Replicate Integration
- **FLUX.1 Model**: Generate appetizing dish images
- Polls for completion with timeout handling
- Fallback to placeholder if generation fails

## Development Guidelines

### Code Organization
- **Single File Backend**: All code in `main.go` as specified
- **Component Structure**: Single React component in `App.jsx`
- **Minimal Dependencies**: Essential packages only

### Error Handling
- Structured error responses with codes and messages
- Graceful degradation for optional features (images)
- Retry logic for transient failures

### Performance Considerations
- Concurrent dish processing with semaphore limiting
- Database connection pooling via GORM
- Image optimization (WebP format, 512x512 size)

## Production Deployment

### Environment Configuration
- Set `DB_SSL_MODE=require` for production databases
- Use environment variables for all secrets
- Configure CORS for your frontend domain

### Scaling Considerations
- Horizontal scaling possible with external queue (Redis/RabbitMQ)
- Database connection pooling for concurrent requests
- CDN for generated dish images

### Monitoring
- Structured JSON logging with Zap
- Health check endpoint at `/health`
- Request tracing with menu/dish IDs

## Cost Optimization

### API Usage
- **OpenAI**: ~$0.01-0.05 per menu (vision + descriptions)
- **Replicate**: ~$0.10-0.30 per menu (image generation)
- Target: <$0.50 per menu including infrastructure

### Optimization Strategies
- Cache generated images in CDN/object storage
- Batch API requests where possible
- Implement request deduplication by image hash

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   - Check PostgreSQL is running
   - Verify connection parameters in `.env`
   - Ensure database exists

2. **API Key Errors**
   - Verify OpenAI/Replicate API keys are valid
   - Check account quotas and billing

3. **Image Generation Timeouts**
   - Replicate API can be slow during peak hours
   - Images may fail but descriptions will still work

4. **CORS Issues**
   - Backend allows all origins in development
   - Configure specific origins for production

### Logs
Check backend logs for detailed error information:

```bash
# Backend logs show structured JSON with context
{"level":"error","msg":"Failed to generate image","dishID":"uuid","error":"timeout"}
```

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes following the existing patterns
4. Test with real menu images
5. Submit a pull request

## Support

For issues and questions:
- Check the troubleshooting section
- Review API documentation
- Open an issue with menu image examples (anonymized)