# Stage 1: Build the application
FROM node:24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy package.json and package-lock.json (or yarn.lock) for dependency installation
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy the rest of the application code
COPY . .

# Build the Next.js app
RUN npm run build

# Stage 2: Serve the application with a smaller, production-ready image
FROM node:24-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy only the necessary files from the builder stage
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/.next ./.next

# Install only production dependencies
RUN npm install --production

# Expose the port the app will run on
EXPOSE 3000

# Start the Next.js app
CMD ["npm", "start"]
