# Use Node.js Alpine for a lightweight base
FROM node:20-alpine

# Set the working directory inside the container
WORKDIR /app

# Install pnpm globally (as the project uses pnpm-lock.yaml)
RUN npm install -g pnpm

# Copy package files first to leverage Docker layer caching
COPY package.json pnpm-lock.yaml* ./

# Install dependencies
RUN pnpm install

# Copy the rest of the application source code
COPY . .

# Build the TypeScript code
RUN pnpm run build

# Expose the port defined in your docker-compose.yaml and .env
EXPOSE 3002

# Start the compiled application
CMD ["pnpm", "run", "dev"]
