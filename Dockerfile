FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /snutt-proxy .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /snutt-proxy /snutt-proxy
EXPOSE 8080
ENTRYPOINT ["/snutt-proxy"]
