ARG ENABLE_PROXY=false
ARG BASE_IMAGE_TAG=nonroot
ARG ADC_VERSION=0.17.0
ARG TARGETARCH

FROM golang:1.22 AS builder
WORKDIR /workspace
COPY go.* ./

RUN if [ "$ENABLE_PROXY" = "true" ] ; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build make build && mv bin/api7-ingress-controller /bin && rm -rf /workspace

RUN wget https://github.com/api7/adc/releases/download/v${ADC_VERSION}/adc-v${ADC_VERSION}-linux-${TARGETARCH}.tar.gz \
    && tar -zxvf adc-v${ADC_VERSION}-linux-${TARGETARCH}.tar.gz \
    && mv adc-v${ADC_VERSION}-linux-${TARGETARCH}/adc /bin \
    && rm -rf adc-v${ADC_VERSION}-linux-${TARGETARCH}.tar.gz adc-v${ADC_VERSION}-linux-${TARGETARCH}

FROM gcr.io/distroless/static-debian12:${BASE_IMAGE_TAG}
WORKDIR /app

COPY --from=builder /bin/api7-ingress-controller .
COPY --from=builder /bin/adc /bin/adc
COPY config/samples/config.yaml ./conf/config.yaml

ENTRYPOINT ["/app/api7-ingress-controller"]
CMD ["-c", "/app/conf/config.yaml"]
