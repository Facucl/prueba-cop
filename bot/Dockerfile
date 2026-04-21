FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bot /bot
USER nonroot:nonroot
ENTRYPOINT ["/bot"]
