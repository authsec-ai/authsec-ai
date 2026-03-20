# Stage 1: Build the React app
FROM node:20-alpine AS builder
WORKDIR /app

# Set environment variables for native module compatibility
# ENV ESBUILD_BINARY_PATH=/usr/local/bin/esbuild
# ENV NPM_CONFIG_UNSAFE_PERM=true
# ENV NPM_CONFIG_LEGACY_PEER_DEPS=true
# ARG VITE_API_URL
# ARG VITE_APP_NAME
# ENV VITE_API_URL=${VITE_API_URL}
# ENV VITE_APP_NAME=${VITE_APP_NAME}

# Copy package.json only (not package-lock.json to avoid platform mismatch)
COPY . .
RUN npm install
RUN npm run build

# Stage 2: Set up the runtime environment
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/server.js .
COPY --from=builder /app/server.json ./package.json
RUN npm install --production \
 && chown -R 1000:1000 /app
 # 🔒 Switch to non-root user
USER 1000

EXPOSE 3000
CMD ["node", "server.js"]
