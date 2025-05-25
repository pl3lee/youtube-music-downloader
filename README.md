# YouTube Music Downloader

## Motivation

This project provides a simple way to download music locally from YouTube Music. It offers a web interface and an API to submit YouTube Music links for download. The primary goal is to have a personal, self-hosted solution for archiving music tracks.

## Features

*   **Web UI:** A simple interface to paste YouTube Music links and initiate downloads.
*   **API Endpoint:** A `/api/download` endpoint to programmatically submit links.
*   **Authentication:** Optional password protection for the API and UI.
*   **Bulk Downloads:** Submit multiple links at once.
*   **Dockerized:** Easy to deploy using Docker.

## Technologies Used

*   **Backend:** Go
*   **CLI Downloader:** `gytmdl` (implicitly used via `exec.Command`)
*   **Containerization:** Docker
*   **CI/CD:** GitHub Actions

## Getting Started

### Prerequisites

*   Go (for local development)
*   Docker (for running as a container)
*   `gytmdl` installed and in your PATH if running locally without Docker. (Note: The Docker container includes this).

### Configuration

The application can be configured using environment variables:

*   `PORT`: The port on which the server will listen (default: `3000`).
*   `PASSWORD`: An optional password to protect the download endpoint. If not set, authentication is disabled.

You can create a `.env` file in the root directory to set these variables for local development:

```env
PORT=3000
PASSWORD=yoursecurepassword
```

### Running Locally

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd <repository-directory>
    ```
2.  **Install dependencies (if any, Go modules will handle this):**
    ```bash
    go mod tidy
    ```
3.  **Run the application:**
    ```bash
    go run main.go
    ```
    The server will start, and you can access the UI at `http://localhost:PORT` (e.g., `http://localhost:3000`).

### Running with Docker

1.  **Build the Docker image:**
    ```bash
    docker build -t youtube-music-downloader .
    ```
2.  **Run the Docker container:**
    ```bash
    docker run -d -p 9005:3000 \
      -e PASSWORD=yoursecurepassword \
      -v $(pwd)/Music:/app/Music \
      --name ytm-downloader \
      youtube-music-downloader
    ```
    *   Replace `yoursecurepassword` with your desired password.
    *   The `-v $(pwd)/Music:/app/Music` flag mounts a local directory named `Music` into the container's `/app/Music` directory, where downloaded files will be saved. Adjust the host path `$(pwd)/Music` as needed.
    *   You can access the UI at `http://localhost:9005`.

    Alternatively, you can use the pre-built images from Docker Hub:
    ```bash
    docker run -d -p 9005:3000 \
      -e PASSWORD=yoursecurepassword \
      -v $(pwd)/Music:/app/Music \
      --name ytm-downloader \
      pl3lee/youtube-music-downloader:latest
    ```

## API Usage

*   **Endpoint:** `POST /api/download`
*   **Authentication:** If a `PASSWORD` is set, include it in the `Authorization` header.
*   **Request Body (JSON):**
    ```json
    {
      "links": ["<youtube-music-link-1>", "<youtube-music-link-2>"]
    }
    ```
*   **Success Response (200 OK):**
    ```json
    {
      "results": [
        {
          "link": "<youtube-music-link-1>",
          "status": "success",
          "error": ""
        },
        {
          "link": "<youtube-music-link-2>",
          "status": "fail",
          "error": "details about the failure"
        }
      ]
    }
    ```
*   **Error Responses:**
    *   `400 Bad Request`: Invalid input or missing links.
    *   `401 Unauthorized`: Missing or incorrect password.
    *   `405 Method Not Allowed`: If not using POST.
    *   `500 Internal Server Error`: Server-side issues.

### Example cURL Request:

```bash
curl -X POST http://localhost:3000/api/download \
-H "Content-Type: application/json" \
-H "Authorization: yoursecurepassword" \
-d '{
  "links": ["https://music.youtube.com/watch?v=xxxxxxxxxxx"]
}'
```

## Docker Hub

Automated builds are pushed to Docker Hub: `pl3lee/youtube-music-downloader`

Available tags:
* `latest`
* Specific commit SHAs

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request or open an Issue.

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

## License

This project is licensed under the MIT License - see the `LICENSE` file for details (if you add one).
