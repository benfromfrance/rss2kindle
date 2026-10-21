# Étape 1 : Compilation du frontend React
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Étape 2 : Compilation du binaire Go avec frontend embarqué
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app/backend

# Dépendances et certificats CA
RUN apk add --no-cache git ca-certificates tzdata

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Copie des fichiers compilés du frontend dans le dossier embed
COPY --from=frontend-builder /app/backend/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/rss2kindle ./cmd/rss2kindle

# Étape 3 : Image finale minimale et sécurisée
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Création d'un utilisateur non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=backend-builder /app/rss2kindle /app/rss2kindle

# Dossier des données persistantes
RUN mkdir -p /data/epub && chown -R appuser:appgroup /data

USER appuser

ENV R2K_ADDR=":8080" \
    R2K_DATA_DIR="/data"

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/app/rss2kindle"]
