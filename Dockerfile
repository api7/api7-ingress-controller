ARG ENABLE_PROXY=false
ARG BASE_IMAGE_TAG=nonroot

FROM golang:1.22 AS builder
WORKDIR /workspace
COPY go.* ./

RUN if [ "$ENABLE_PROXY" = "true" ] ; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build make build && mv bin/api7-ingress-controller /bin && rm -rf /workspace

FROM gcr.io/distroless/static-debian12:${BASE_IMAGE_TAG}
WORKDIR /app

COPY --from=builder /bin/api7-ingress-controller .
COPY conf/config.yaml ./conf/config.yaml

ENTRYPOINT ["/app/api7-ingress-controller"]
CMD ["-c", "/app/conf/config.yaml"]
