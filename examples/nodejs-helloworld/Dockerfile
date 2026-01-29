FROM node:20-alpine

WORKDIR /app

# Copy package file first for better layer caching
COPY package.json ./

# Install dependencies (none for this minimal app; keeps Dockerfile reusable)
RUN npm install --omit=dev

# Copy application code
COPY server.js ./

# Expose the server port
EXPOSE 3000

# Run the server
CMD ["node", "server.js"]
