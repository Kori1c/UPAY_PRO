FROM node:20-alpine AS frontend-builder

WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build


FROM golang:1.25-alpine AS backend-builder

WORKDIR /src
RUN apk add --no-cache git build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=frontend-builder /static/admin_spa ./static/admin_spa

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/upay_pro .


FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=backend-builder /out/upay_pro ./upay_pro
COPY --from=backend-builder /src/static ./static
COPY --from=backend-builder /src/plugins ./plugins

RUN mkdir -p /app/DBS /app/logs

ENV TZ=Asia/Shanghai
ENV UPAY_HTTP_PORT=8090
ENV UPAY_APP_URL=http://localhost:8090
ENV UPAY_REDIS_HOST=redis
ENV UPAY_REDIS_PORT=6379
ENV UPAY_REDIS_DB=0

EXPOSE 8090
VOLUME ["/app/DBS", "/app/logs"]

CMD ["./upay_pro"]
