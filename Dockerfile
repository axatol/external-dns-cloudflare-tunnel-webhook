FROM golang:1 AS build
WORKDIR /go/src/app
COPY . .
ARG BUILD_COMMIT=unknown
ARG BUILD_TIME=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
  make build-binary BUILD_COMMIT=${BUILD_COMMIT} BUILD_TIME=${BUILD_TIME} OUTPUT=/go/bin/app

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /go/bin/app /app
CMD ["/app"]
