# Container Deployment Guide

This document explains how to properly deploy the Atlas UI application in containerized environments to avoid common image optimization issues.

## Environment Variables

### Required for Container Environments

Set one of the following environment variables to enable container-optimized image handling:

```bash
# For Docker containers
DOCKER_ENV=true

# Alternative: Kubernetes environments are auto-detected via KUBERNETES_SERVICE_HOST
# Alternative: Force disable image optimization
DISABLE_IMAGE_OPTIMIZATION=true
```

## Docker Deployment

### Dockerfile Example

```dockerfile
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm ci

# Copy source code
COPY . .

# Set environment variables
ENV NODE_ENV=production
ENV DOCKER_ENV=true

# Build the application
RUN npm run build

EXPOSE 3000

CMD ["npm", "start"]
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  atlas-ui:
    build: .
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - DOCKER_ENV=true
      - NEXT_PUBLIC_API_URL=http://localhost:8080
    restart: unless-stopped
```

## Kubernetes Deployment

Kubernetes environments are automatically detected via the `KUBERNETES_SERVICE_HOST` environment variable that Kubernetes injects automatically.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-ui
spec:
  replicas: 2
  selector:
    matchLabels:
      app: atlas-ui
  template:
    metadata:
      labels:
        app: atlas-ui
    spec:
      containers:
      - name: atlas-ui
        image: atlas-ui:latest
        ports:
        - containerPort: 3000
        env:
        - name: NODE_ENV
          value: "production"
        - name: NEXT_PUBLIC_API_URL
          value: "https://api.atlas.example.com"
```

## Troubleshooting

### Image Loading Issues

If you're still experiencing 400 errors with images in containers:

1. **Check Environment Variables**: Ensure `DOCKER_ENV=true` is set
2. **Verify Container Detection**: The app automatically detects containers in production
3. **External Images**: All external images (maplestory.io) are automatically unoptimized
4. **SVG Files**: All SVG files are automatically unoptimized

### Debug Mode

To debug image loading in containers, you can check the environment detection:

```javascript
// In browser console or server logs
console.log('Container Environment:', process.env.DOCKER_ENV === 'true');
console.log('Kubernetes Environment:', process.env.KUBERNETES_SERVICE_HOST !== undefined);
console.log('Production Mode:', process.env.NODE_ENV === 'production');
```

### Performance Considerations

In container environments:
- Images are served unoptimized to avoid Next.js optimization issues
- External images (maplestory.io API) bypass optimization entirely  
- SVG files are never optimized
- Image loading strategy defaults to 'eager' for better reliability

This approach trades some optimization for reliability in containerized deployments.