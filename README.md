# Moodify 🎵🧠

> **AI-Powered Multimodal Music Intelligence, Continuous Vector Mood Similarity & Cloud-Native Playlist Engineering**  
> *Built for the AWS Hackathon 2026*

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Python Version](https://img.shields.io/badge/Python-3.11+-3776AB?style=for-the-badge&logo=python)](https://python.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16%2B%20pgvector-336791?style=for-the-badge&logo=postgresql)](https://github.com/pgvector/pgvector)
[![AWS Powered](https://img.shields.io/badge/AWS-S3%20%7C%20SQS%20%7C%20ECS%20%7C%20RDS-FF9900?style=for-the-badge&logo=amazon-aws)](https://aws.amazon.com/)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](./LICENSE)

---

## 🌟 Overview

**Moodify** is a high-performance audio intelligence platform designed to ingest music libraries at scale, extract acoustic digital signal processing (DSP) features alongside deep contextual lyrical sentiment, and construct **64-dimensional continuous multimodal mood embeddings** stored in PostgreSQL using `pgvector`.

By fusing 36-D low-level acoustic descriptors (spectral centroid, chroma, MFCCs, tonnetz, rhythm) with 28-D RoBERTa-based lyrical emotion distributions (trained on GoEmotions), Moodify completely eliminates arbitrary keyword heuristics and if/else sentiment rules. Songs are indexed in high-dimensional vector space, enabling sub-millisecond nearest-neighbor mood chaining, intelligent playlist transitions, and smooth BPM sequencing.

Moodify is built cloud-native for **Amazon Web Services (AWS)**, decoupling storage to **Amazon S3**, distributing heavy DSP/NLP analysis across **Amazon SQS** queues and worker pools, and maintaining fault-tolerant local fallback adapters for seamless local development and hackathon demonstrations.

---

## 🏗️ AWS Cloud Deployment Architecture

Moodify implements an enterprise-grade, decoupled microservices architecture on AWS designed for high-concurrency audio ingestion, compute-heavy background processing, and low-latency vector similarity retrieval.

```
                                      ┌────────────────────────────────────────────────────────┐
                                      │                   AWS Virtual Private Cloud (VPC)      │
                                      │                                                        │
┌───────────────────────────────┐     │   ┌────────────────────────────────────────────────┐   │
│ Client / Modern Web Frontend  │─────┼──>│       Application Load Balancer (ALB)          │   │
│  (Vite + 3D Mood Galaxy)      │     │   └───────────────────────┬────────────────────────┘   │
└───────────────────────────────┘     │                           │                            │
                                      │                           ▼                            │
                                      │   ┌────────────────────────────────────────────────┐   │
                                      │   │  AWS ECS Fargate: Moodify API Service (Go)     │   │
                                      │   │   - Streaming Chunked Multipart Ingest         │   │
                                      │   │   - AcoustID / MusicBrainz Fingerprinting      │   │
                                      │   │   - Dynamic M3U8 Playlist Chaining Engine      │   │
                                      │   └───────┬───────────────────────────────┬────────┘   │
                                      │           │                               │            │
                                      │           │ (Audio Stream)                │ (Enqueue)  │
                                      │           ▼                               ▼            │
                                      │   ┌───────────────┐              ┌────────────────┐   │
                                      │   │   Amazon S3   │              │   Amazon SQS   │   │
                                      │   │ Audio Storage │              │ Analysis Queue │   │
                                      │   └───────┬───────┘              └────────┬───────┘   │
                                      │           │                               │            │
                                      │           │ (Fetch Track)                 │ (Poll Job) │
                                      │           ▼                               ▼            │
                                      │   ┌────────────────────────────────────────────────┐   │
                                      │   │  AWS ECS Fargate / AWS Batch Worker Pool       │   │
                                      │   │   - Librosa 36-D Acoustic DSP Extraction       │   │
                                      │   │   - RoBERTa 28-D Continuous Lyrical NLP       │   │
                                      │   │   - 64-D Multimodal Normalization & Fusion     │   │
                                      │   └───────────────────────┬────────────────────────┘   │
                                      │                           │                            │
                                      │                           │ (Upsert Vectors)           │
                                      │                           ▼                            │
                                      │   ┌────────────────────────────────────────────────┐   │
                                      │   │  Amazon RDS Aurora PostgreSQL (Multi-AZ)       │   │
                                      │   │   - pgvector HNSW Index (Cosine Similarity)    │   │
                                      │   │   - Relational Metadata, Lyrics & Sessions     │   │
                                      │   └────────────────────────────────────────────────┘   │
                                      └────────────────────────────────────────────────────────┘
```

### Architectural Highlights

| AWS Component | Service Role & Design Justification |
| :--- | :--- |
| **Amazon S3** | **Decoupled Audio Object Storage**: Direct multi-part streaming ingestion offloads heavy audio bytes from application containers. Integrated with S3 Lifecycle policies for temporary session cleanups. |
| **Amazon SQS** | **Asynchronous Job Offloading**: Buffers batch analysis requests (`POST /api/v1/songs/batch/analyze`) into lightweight JSON jobs. Isolates bursty library uploads from compute-heavy neural inference. |
| **AWS ECS Fargate (API)** | **High-Throughput Go Ingress**: Containerized Go 1.22 Chi service running non-blocking asynchronous I/O with automatic horizontal autoscaling (CPU/Memory thresholds). |
| **AWS ECS / AWS Batch (Workers)** | **Scalable Audio DSP & NLP Workers**: Auto-scales based on `ApproximateNumberOfMessagesVisible` in SQS. Executes Librosa feature extraction and transformer inference in containerized sidecars. |
| **Amazon RDS (PostgreSQL + pgvector)** | **High-Dimensional Vector Engine**: Stores and indexes 64-D normalized multimodal embeddings using **HNSW** (`m=16, ef_construction=64`), providing logarithmic search latency for mood neighbor queries. |
| **AWS Application Load Balancer** | **SSL Termination & Traffic Routing**: Handles health checks (`/api/v1/health`), connection pooling, and distributes client traffic across availability zones. |
| **AWS Secrets Manager & IAM** | **Zero-Credential Codebase**: Uses IAM Roles for Tasks (IRSA) to grant fine-grained permissions for S3 read/write and SQS enqueue/dequeue without hardcoded API keys. |

---

## 🧠 Multimodal AI & Signal Processing Pipeline

Moodify does **not** rely on rule-based keyword matching, static sentiment dictionaries, or arbitrary heuristics. Every song is mapped into a continuous 64-dimensional space combining auditory physics and linguistic semantics:

```
                            ┌──────────────────────────────────────────────┐
                            │               Audio Track (.mp3)             │
                            └──────────────────────┬───────────────────────┘
                                                   │
                         ┌─────────────────────────┴─────────────────────────┐
                         ▼                                                   ▼
         ┌───────────────────────────────┐                   ┌───────────────────────────────┐
         │     Acoustic DSP Engine       │                   │    Lyrics Analysis Engine     │
         │          (Librosa)            │                   │      (RoBERTa GoEmotions)     │
         └───────────────┬───────────────┘                   └───────────────┬───────────────┘
                         │                                                   │
          [36 Continuous DSP Features]                        [28 Continuous Emotion Probs]
          • 13 MFCCs (Timbral texture)                        • Admiration, Amusement, Anger
          • 12 Chroma (Harmonic content)                      • Joy, Optimism, Sadness, Grief
          • 7 Spectral Contrast (Dynamics)                    • Love, Desire, Remorse, Fear
          • 2 Tonnetz (Tonal centroid)                        • 28 Dimensional continuous
          • Tempo / BPM & RMS Energy                            probability distribution
                         │                                                   │
                         └─────────────────────────┬─────────────────────────┘
                                                   │
                                                   ▼
                                 ┌───────────────────────────────────┐
                                 │    Multimodal Fusion Layer        │
                                 │   - L2 Unit Normalization         │
                                 │   - Harmonic Concatenation        │
                                 └─────────────────┬─────────────────┘
                                                   │
                                                   ▼
                                 ┌───────────────────────────────────┐
                                 │     64-D Unified Mood Vector      │
                                 │  Indexed via pgvector (HNSW)      │
                                 └───────────────────────────────────┘
```

### Dynamic Compound Mood Classification
Instead of forcing songs into generic buckets ("Happy" vs "Sad"), Moodify computes the top two dominant continuous lyrical emotion states and couples them with acoustic valence and energy:
- **`joy & optimism`** (High energy, bright tonality)
- **`sadness & grief`** (Low energy, acoustic minor tonality)
- **`desire & love`** (Warm harmonics, mid-tempo groove)
- **`fear & nervousness`** (High spectral contrast, irregular rhythm)

---

## 🚀 Key Features

- 📂 **Massive Library Batch Ingest**: Streamed multipart ingestion (`POST /api/v1/songs/batch/upload`) parses files directly to storage without memory bloat.
- 🛑 **Strict File Ceiling**: Configurable per-track limit (**35 MB default**) via `MAX_FILE_SIZE_MB` prevents denial-of-service and runaway storage consumption.
- ⚡ **Asynchronous Analysis Queue**: Fire-and-forget batch analysis (`POST /api/v1/songs/batch/analyze`) responds with `202 Accepted` and offloads work to AWS SQS / Worker pools.
- 🎼 **Seed-Based Playlist Chaining**: Generates intelligent continuous playlists (`POST /api/v1/playlists/generate`) by traversing nearest neighbors in 64-D mood vector space, sorting by smooth BPM transitions, and exporting ready-to-play `.m3u8` playlists.
- 🔍 **Acoustic Fingerprinting**: Automated audio identification via `fpcalc` and AcoustID / MusicBrainz metadata tagging.
- 📜 **Synchronized Lyrics Fetching**: Automatic retrieval of line-by-line synchronized `.lrc` lyrics via LRCLIB.
- 🔄 **Cloud-First with Zero-Downtime Fallback**: Integrated `S3Store` and `SQSQueue` fall back seamlessly to local disk and worker pools when AWS credentials are offline.

---

## 📡 REST API Reference

All endpoints are versioned under `/api/v1`.

### Song & Ingestion Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/songs/upload` | Upload a single audio file (MP3, M4A, FLAC, WAV, OGG). |
| `POST` | `/songs/batch/upload` | Stream upload multiple audio files in a single request. |
| `GET` | `/songs` | List songs with pagination, metadata, and mood status. |
| `GET` | `/songs/{id}/download` | Stream audio file with Range support or download attachment. |
| `DELETE` | `/songs/{id}` | Delete song metadata, vector records, and physical file. |

### Analysis & Vector Intelligence Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/songs/{id}/analyze` | Trigger synchronous acoustic DSP & lyrical mood analysis. |
| `POST` | `/songs/batch/analyze` | Enqueue unanalyzed songs to AWS SQS / worker pool (`202 Accepted`). |
| `GET` | `/songs/batch/status` | Get real-time queue depth and processing statistics. |
| `GET` | `/songs/{id}/features` | Retrieve 36-D DSP acoustic features, tempo, and key. |
| `GET` | `/songs/{id}/lyrics` | Retrieve synchronized `.lrc` lyrics and emotion breakdown. |
| `GET` | `/songs/{id}/similar` | Find nearest-neighbor songs using `pgvector` cosine similarity (`mode=audio\|lyrics\|multimodal`). |
| `GET` | `/songs/clusters` | Retrieve library mood clusters grouped by nuanced emotion tags. |

### Metadata, Playlist & Session Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/songs/{id}/identify` | Fingerprint audio with Chromaprint and fetch MusicBrainz tags. |
| `PUT` | `/songs/{id}/metadata` | Update artist, title, album, year, and genre. |
| `POST` | `/songs/{id}/embed` | Write updated metadata tags directly back into audio file ID3 chunks. |
| `POST` | `/playlists/generate` | Generate vector-chained dynamic playlist (`format=json\|m3u8`). |
| `DELETE` | `/sessions/{id}` | Purge all tracks and session state for a given session. |
| `GET` | `/health` | Check API server and database connectivity. |

---

## 🛠️ Quickstart (Local Development)

### 1. Prerequisites
- **Go**: 1.22 or higher
- **Python**: 3.11 or higher
- **Docker & Docker Compose**: For local PostgreSQL + `pgvector`
- **Audio Tools**: `ffmpeg` and `fpcalc` (Chromaprint)
  ```bash
  # Fedora / RHEL
  sudo dnf install ffmpeg chromaprint
  # Ubuntu / Debian
  sudo apt install ffmpeg libchromaprint-tools
  # macOS
  brew install ffmpeg chromaprint
  ```

### 2. Start Local Database
```bash
docker-compose up -d
```
This boots PostgreSQL 16 with the `pgvector` extension enabled on port `5432`.

### 3. Setup Python Virtual Environment
```bash
cd python
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
cd ..
```

### 4. Configure Environment Variables
Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```
Key configuration parameters:
```dotenv
PORT=8080
DATABASE_URL=postgres://moodify:moodify@localhost:5432/moodify?sslmode=disable
UPLOAD_DIR=uploads
MAX_FILE_SIZE_MB=35
ACOUSTID_API_KEY=your_acoustid_api_key_here

# AWS Configuration (Optional - falls back to local storage and worker pool if omitted)
AWS_REGION=us-east-1
AWS_S3_BUCKET=moodify-audio-assets
AWS_SQS_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/123456789012/moodify-analysis-jobs
```

### 5. Run the Go API Server
```bash
go run ./cmd/server
```
The server will start on `http://localhost:8080`.

### 6. Launch Modern Web Frontend
In another terminal:
```bash
cd frontend
npm install
npm run dev
```
Open `http://localhost:5173` to explore the interactive 3D Mood Galaxy, upload your audio library, and generate playlists.

---

## ☁️ Deploying to AWS (ECS Fargate Production Guide)

Moodify uses an optimized multi-container task architecture in AWS ECS Fargate (`awsvpc` networking), persistent audio storage on Amazon EFS, and Amazon RDS Aurora PostgreSQL with `pgvector`.

For the complete end-to-end guide, see [deploy/AWS_ECS_DEPLOYMENT.md](file:///home/ever/Code/Projects/Moodify/deploy/AWS_ECS_DEPLOYMENT.md).

### 1. Build and Push Multi-Container Images to Amazon ECR

```bash
export AWS_REGION="us-east-1"
export ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Authenticate Docker to ECR
aws ecr get-login-password --region $AWS_REGION | \
  docker login --username AWS --password-stdin ${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# 1. Build & Push Unified Backend (Go 1.22 + Python 3.11 DSP / RoBERTa)
docker build --platform linux/amd64 -t moodify-backend:latest -f Dockerfile .
docker tag moodify-backend:latest ${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/moodify-backend:latest
docker push ${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/moodify-backend:latest

# 2. Build & Push Frontend (Vite React SPA + Nginx Reverse Proxy)
docker build --platform linux/amd64 -t moodify-frontend:latest -f frontend/Dockerfile frontend/
docker tag moodify-frontend:latest ${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/moodify-frontend:latest
docker push ${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/moodify-frontend:latest
```

### 2. Task Definition & Hardware Requirements

The pre-configured task definition is available in [deploy/ecs-task-definition.json](file:///home/ever/Code/Projects/Moodify/deploy/ecs-task-definition.json).

* **Memory Ceiling**: Specify at least **1 vCPU (1024)** and **4 GB RAM (4096)**. Neural transformer loading (PyTorch + RoBERTa) requires a minimum 3.2 GB working memory ceiling to avoid exit code 137 (OOM-Killed).
* **Storage**: Mount an **Amazon EFS** volume to `/app/uploads` so audio files persist across Fargate task restarts.
* **Networking**: In `awsvpc` mode, the Nginx frontend reverse-proxies `/api/v1/` to the backend over `127.0.0.1:8080`.
* **Health Check**: Configured via `python3` urllib health probe against `http://127.0.0.1:8080/api/v1/health`.


---

## 🧪 Testing

Moodify enforces comprehensive unit and integration testing across all Go packages:
```bash
# Run all tests with race detector and verbose logging
go test -v -race ./...

# Test queue and background worker serialization
go test -v ./internal/queue

# Test multimodal vector fusion and genre inference
go test -v ./internal/fusion
```

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
