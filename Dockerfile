FROM node:22-alpine AS frontend
WORKDIR /app/client
RUN corepack enable
COPY client/package.json client/pnpm-lock.yaml client/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY client/ ./
RUN pnpm build

FROM golang:1.26 AS backend
WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=frontend /app/client/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /server ./cmd/main.go

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=backend /server ./server
COPY server/prompts ./prompts
CMD ["./server"]

