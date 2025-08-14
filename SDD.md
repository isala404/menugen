
# Software Design Document: Menu Visualizer

## 1. Introduction

The Menu Visualizer is a proof-of-concept (POC) application designed to digitize and enhance restaurant menus. Users can upload a photograph of a physical menu. The system then uses AI to extract the menu's text, identify dishes, and generate illustrative images for each dish. The result is an interactive, visually appealing digital menu.

This document outlines the technical design, architecture, and components of the Menu Visualizer application.

## 2. System Architecture

The application is composed of three main components: a React frontend, a Go backend, and a PostgreSQL database.

```
+-----------------+      (1) Upload Menu Image      +-----------------+
|                 |-------------------------------->|                 |
|  React Frontend |                                 |   Go Backend    |
| (Plain CSS)     |<--------------------------------| (Single File)   |
|                 |   (4) Display Menu & Images     |                 |
+-----------------+                                 +-------+---------+
                                                            |
                                                            | (2) Extract Menu (GPT-4o)
                                                            | (3) Generate Images (Replicate)
                                                            |
                                          +-----------------+-----------------+
                                          |                 |                 |
                                  +-------v---------+  +----v-----+   +-------v-------+
                                  |   OpenAI API    |  | Replicate|   |   PostgreSQL  |
                                  |    (GPT-4o)     |  |    API   |   |   (gorm)      |
                                  +-----------------+  +----------+   +---------------+
```

### Data Flow:
1.  **Menu Upload:** The user uploads a menu image via the React frontend.
2.  **Backend Processing:** The Go backend receives the image.
    - It calls the OpenAI GPT-4o API to extract structured menu data (in XML format).
    - For each dish extracted, it calls the Replicate API to generate an image. This is an asynchronous process that involves polling for the result.
3.  **Database Storage:** Once the data is processed and images are generated, the backend stores the menu text and image blobs in the PostgreSQL database using `gorm`.
4.  **Display:** The React frontend fetches and displays the structured menu, including the newly generated dish images.

## 3. Frontend (React App)

The frontend is a single-page application (SPA) built with React and styled with plain CSS for simplicity.

-   **Responsibilities:**
    -   Provide an interface for users to upload a menu image.
    -   Display the status of the menu processing (e.g., "Extracting dishes...", "Generating images...").
    -   Fetch and render the final, visualized menu from the backend.
    -   Display dish names, descriptions, prices, and the AI-generated images.

-   **Component Breakdown:**
    -   `App.js`: The main component that manages state and renders other components.
    -   `MenuUploader.js`: A component with a form for file uploads. Handles the POST request to the backend.
    -   `MenuDisplay.js`: A component to display the structured menu. It will map over the dish data and render `DishItem` components.
    -   `DishItem.js`: A component to display a single dish with its name, description, price, and the generated image.

-   **Key Libraries:**
    -   `react`: For building the user interface.
    -   `axios` (or `fetch` API): For making HTTP requests to the backend.

## 4. Backend (Go)

The backend is a single Go file that serves a RESTful API.

-   **Responsibilities:**
    -   Expose an API endpoint for menu image uploads.
    -   Handle multipart/form-data for the image upload.
    -   Communicate with the OpenAI API to parse the menu.
    -   Communicate with the Replicate API for image generation, including the polling logic to get completed images.
    -   Connect to the PostgreSQL database and perform CRUD operations using `gorm`.

-   **API Endpoints:**
    -   `POST /api/menu`: Accepts a menu image upload. It will process the menu and return a menu ID.
    -   `GET /api/menu/{id}`: Returns the structured menu data and image URLs for a given menu ID.

-   **Go Backend Logic Flow:**
    1.  Receive an image file on the `/api/menu` endpoint.
    2.  Encode the image to a suitable format (e.g., base64) to send to GPT-4o.
    3.  Construct a prompt for GPT-4o, asking it to return an XML structure of the menu items.
    4.  Send the request to GPT-4o and parse the resulting XML.
    5.  For each dish in the XML:
        -   Create a new image generation request to the Replicate API with the dish name/description.
        -   Start a polling loop to check the status of the image generation.
        -   Once an image is successfully generated, download the image data (blob).
    6.  Save the menu and all dish details (text and image blobs) to the PostgreSQL database.
    7.  Return a success response to the client.

-   **Key Libraries:**
    -   `net/http`: For creating the web server and API endpoints.
    -   `gorm.io/gorm`: ORM for PostgreSQL.
    -   `gorm.io/driver/postgres`: PostgreSQL driver for `gorm`.
    -   Official Go clients for OpenAI and Replicate, or standard `http` package for manual API calls.

## 5. Database (PostgreSQL)

The database will store the menu information and the generated images.

-   **ORM:** `gorm` will be used to map Go structs to database tables.

-   **Database Schema:**

    **`menus` table:**
    | Column Name | Data Type | Constraints | Description |
    | --- | --- | --- | --- |
    | `id` | `bigserial` | `PRIMARY KEY` | Unique identifier for the menu. |
    | `created_at` | `timestamptz` | | Timestamp of creation. |
    | `restaurant_name` | `text` | | Name of the restaurant (optional, extracted by AI). |

    **`dishes` table:**
    | Column Name | Data Type | Constraints | Description |
    | --- | --- | --- | --- |
    | `id` | `bigserial` | `PRIMARY KEY` | Unique identifier for the dish. |
    | `menu_id` | `bigint` | `FOREIGN KEY (menus.id)` | Associates the dish with a menu. |
    | `name` | `text` | `NOT NULL` | Name of the dish. |
    | `description` | `text` | | Description of the dish. |
    | `price` | `varchar(20)` | | Price of the dish. |
    | `image_blob` | `bytea` | | The binary data of the generated image. |
    | `created_at` | `timestamptz` | | Timestamp of creation. |

## 6. Assumptions and Constraints

-   **API Keys:** The application assumes that valid API keys for OpenAI and Replicate are available as environment variables.
-   **Single File Backend:** The Go backend is constrained to a single file for simplicity, which may not be ideal for a production application.
-   **Polling:** The Replicate API requires polling. The implementation will need to handle this, potentially with timeouts.
-   **Image Storage:** Storing images as blobs (`bytea`) in the database is suitable for a POC but may not be the most performant or cost-effective solution for a large-scale application (where a dedicated object store like S3 would be better).

## 7. Future Improvements

-   **Object Storage:** Use a cloud storage solution like AWS S3 or Google Cloud Storage for images instead of storing blobs in the database.
-   **Websockets:** Implement websockets to provide real-time updates to the frontend as the menu processing and image generation progresses, eliminating the need for the client to poll for results.
-   **Backend Refactoring:** Split the single-file Go backend into a more structured project layout (e.g., using separate packages for handlers, models, and services).
-   **Error Handling:** Implement more robust error handling and user feedback mechanisms.
-   **Scalability:** Introduce a background job queue (e.g., RabbitMQ or Redis) to handle the AI processing tasks asynchronously, making the API more responsive.
