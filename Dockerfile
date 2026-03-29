# Stage 1: Build React frontend
FROM node:20-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o conductor ./cmd/conductor

# Stage 3: Final image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/conductor .
COPY --from=web-builder /app/web/dist ./web/dist
COPY migrations/ ./migrations/
EXPOSE 3100 2222
CMD ["/app/conductor"]
