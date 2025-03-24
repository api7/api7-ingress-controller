ARG ENABLE_PROXY=false
ARG BASE_IMAGE_TAG=nonroot

FROM debian:bullseye-slim AS deps
WORKDIR /workspace

ARG ADC_VERSION=0.18.0
ARG TARGETARCH

RUN apt update \
    && apt install -y wget \
    && wget https://github.com/api7/adc/releases/download/v${ADC_VERSION}/adc_${ADC_VERSION}_linux_${TARGETARCH}.tar.gz -O adc.tar.gz \
    && tar -zxvf adc.tar.gz \
    && mv adc /bin/adc \
    && rm -rf adc.tar.gz \
    && apt autoremove -y wget

FROM gcr.io/distroless/static-debian12:${BASE_IMAGE_TAG}
WORKDIR /app

COPY --from=deps /bin/adc /bin/adc
COPY ./bin/api7-ingress-controller .
COPY ./config/samples/config.yaml ./conf/config.yaml

ENTRYPOINT ["/app/api7-ingress-controller"]
CMD ["-c", "/app/conf/config.yaml"]
