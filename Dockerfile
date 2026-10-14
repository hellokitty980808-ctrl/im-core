FROM golang:1.26.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/user ./app/user/service/cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/chat ./app/chat/service/cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/connect ./app/connect/service/cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/gateway ./app/gateway/cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/ /app/bin/
COPY app/user/service/configs/ /app/configs/user/
COPY app/chat/service/configs/ /app/configs/chat/
COPY app/connect/service/configs/ /app/configs/connect/
COPY app/gateway/configs/ /app/configs/gateway/
CMD ["/app/bin/gateway", "-conf", "/app/configs/gateway"]
