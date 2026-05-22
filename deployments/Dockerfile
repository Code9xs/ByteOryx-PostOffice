FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm install --frozen-lockfile || npm install
COPY web/ .
RUN npm run build

FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /postoffice ./cmd/postoffice

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /postoffice /usr/local/bin/postoffice
COPY --from=builder /app/web/dist /var/www/postoffice
EXPOSE 25 587 993 8080
ENTRYPOINT ["postoffice"]
