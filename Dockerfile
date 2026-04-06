FROM node:20-alpine AS frontend
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ .
RUN npm run build

FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o clotho ./cmd/clotho

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=backend /app/clotho /usr/local/bin/clotho
COPY --from=backend /app/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["clotho"]
