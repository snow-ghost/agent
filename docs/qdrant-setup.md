# Qdrant Setup Guide

This guide explains how to set up and use Qdrant vector database with the Agent system.

## Overview

Qdrant is used as the vector database for storing and searching artifact embeddings. The system uses a hybrid approach where:
- Artifact metadata is stored in the filesystem
- Vector embeddings are stored in Qdrant
- Both are indexed for fast retrieval

## Local Installation

### Using Docker (Recommended)

```bash
# Start Qdrant
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant:latest

# Or using docker-compose
make qdrant-up
```

### Using Docker Compose

The system includes Qdrant in the main docker-compose.yml file:

```bash
# Start all services including Qdrant
docker-compose up -d

# Start only Qdrant
docker-compose up -d qdrant
```

## Configuration

### Environment Variables

```bash
# Vector Database
VECTOR_BACKEND=qdrant
QDRANT_URL=http://localhost:6333
QDRANT_COLLECTION=artifacts
QDRANT_DIMENSION=1536
QDRANT_DISTANCE=cosine

# Embeddings
EMBEDDINGS_MODE=lmstudio
LMSTUDIO_BASE_URL=http://localhost:1234
EMBEDDINGS_MODEL=text-embedding-3-small
```

### Collection Schema

The default collection is created with:
- **Name**: `artifacts`
- **Dimension**: 1536 (for text-embedding-3-small)
- **Distance**: Cosine similarity
- **Payload**: Artifact metadata (ID, version, domain, description, etc.)

## Management Commands

### Using Makefile

```bash
# Start Qdrant
make qdrant-up

# Stop Qdrant
make qdrant-down

# View logs
make qdrant-logs

# Get statistics
make qdrant-stats

# Clear collection
make qdrant-clear

# Reindex artifacts
make qdrant-reindex

# Search collection
make qdrant-query QUERY="sorting algorithm"

# Verify consistency
make qdrant-verify
```

### Using KB Manager

```bash
# List artifacts
go run ./cmd/kb-manager -action list

# Get statistics
go run ./cmd/kb-manager -action stats

# Clear collection
go run ./cmd/kb-manager -action clear

# Reindex from filesystem
go run ./cmd/kb-manager -action reindex -artifacts-dir ./artifacts

# Search artifacts
go run ./cmd/kb-manager -action query -query "your search query"

# Verify consistency
go run ./cmd/kb-manager -action verify -artifacts-dir ./artifacts
```

## API Usage

### Direct Qdrant API

```bash
# Health check
curl http://localhost:6333/health

# Get collections
curl http://localhost:6333/collections

# Get collection info
curl http://localhost:6333/collections/artifacts

# Search vectors
curl -X POST http://localhost:6333/collections/artifacts/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, ...],
    "limit": 10,
    "with_payload": true
  }'
```

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```bash
   # Check if Qdrant is running
   docker ps | grep qdrant
   
   # Check logs
   make qdrant-logs
   ```

2. **Collection Not Found**
   ```bash
   # Reindex artifacts
   make qdrant-reindex
   ```

3. **Embedding Errors**
   ```bash
   # Check LMStudio is running
   curl http://localhost:1234/v1/models
   
   # Use mock embeddings for testing
   export EMBEDDINGS_MODE=mock
   ```

4. **Memory Issues**
   ```bash
   # Check Qdrant memory usage
   docker stats qdrant
   
   # Increase Docker memory limits
   ```

### Performance Tuning

1. **Batch Operations**
   - Use `BatchUpsert` for large indexing operations
   - Default batch size is 10, adjust based on memory

2. **Search Optimization**
   - Use appropriate `topK` values
   - Consider payload filtering for large collections

3. **Resource Limits**
   - Monitor Qdrant memory usage
   - Adjust Docker resource limits as needed

## Monitoring

### Health Checks

```bash
# Qdrant health
curl http://localhost:6333/health

# Collection stats
make qdrant-stats

# Consistency check
make qdrant-verify
```

### Metrics

The system exposes Prometheus metrics for:
- Vector search duration
- Search result count
- Indexing operations
- Error rates

## Backup and Recovery

### Backup

```bash
# Backup Qdrant data
docker cp qdrant:/qdrant/storage ./qdrant-backup

# Backup artifacts
tar -czf artifacts-backup.tar.gz ./artifacts
```

### Recovery

```bash
# Restore Qdrant data
docker cp ./qdrant-backup qdrant:/qdrant/storage
docker-compose restart qdrant

# Restore artifacts
tar -xzf artifacts-backup.tar.gz
make qdrant-reindex
```

## Development

### Local Development

```bash
# Start development environment
make dev-setup

# Run tests
make test-all-tasks

# Clean up
make dev-clean
```

### Testing

```bash
# Run integration tests
make docker-test-full

# Run specific test categories
make test-simple
make test-complex
make test-decomposable
```
