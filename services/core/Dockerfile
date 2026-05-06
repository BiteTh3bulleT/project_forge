FROM golang:1.23-bookworm AS build

WORKDIR /src/services/core
COPY services/core/go.mod services/core/go.sum ./
RUN go mod download

COPY services/core/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/forge-core .

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates wget intel-gpu-tools libcap2-bin \
	&& setcap cap_perfmon,cap_sys_admin+ep /usr/bin/intel_gpu_top \
	&& rm -rf /var/lib/apt/lists/* \
	&& groupadd --system forge \
	&& useradd --system --uid 10001 --gid forge --home-dir /app --shell /usr/sbin/nologin forge \
	&& mkdir -p /data /models /workspace \
	&& chown -R forge:forge /data /models /workspace

USER forge
WORKDIR /app

ENV FORGE_CORE_PORT=18492 \
	FORGE_DATA_DIR=/data \
	FORGE_MODEL_HOME=/models \
	FORGE_WORKSPACE_DIR=/workspace

COPY --from=build /out/forge-core /app/forge-core

EXPOSE 18492
VOLUME ["/data", "/models", "/workspace"]

ENTRYPOINT ["/app/forge-core"]
