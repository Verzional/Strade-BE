# ==========================================
# Stage 1: Build the TypeScript code
# ==========================================
FROM node:20-alpine AS builder

WORKDIR /usr/src/app

# Copy package files (this will grab package.json and package-lock.json)
COPY package*.json ./

# Install ALL dependencies
RUN npm install

# Copy source code and build
COPY . .
RUN npm run build

# ==========================================
# Stage 2: Production Environment
# ==========================================
FROM node:20-alpine

WORKDIR /usr/src/app

ENV NODE_ENV=production

# Copy package files
COPY package*.json ./

# Install ONLY production dependencies 
# (Using --omit=dev is the modern npm equivalent of --only=production)
RUN npm install --omit=dev

# Copy compiled code from the builder stage
COPY --from=builder /usr/src/app/dist ./dist

# Expose the port matching your docker-compose config
EXPOSE 3000

# Start the Node app
CMD ["npm", "start"]