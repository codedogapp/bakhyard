# Build from the repo's root: docker build -f services/planner/Dockerfile . (docs/plan.md, decisions 138 and 416)
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine@sha256:fa4881266890ef051f0ce63afdfaccb69bbe8186084387f4fafd60d7ecbabc71 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY internal/ internal/
COPY services/planner/ services/planner/
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/planner ./services/planner

# The Copilot CLI at the version that the Copilot SDK in go.mod names, for linux/amd64 alone;
# mise run copilot:check fails until the version is the SDK's and the checksum the release's.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine@sha256:fa4881266890ef051f0ce63afdfaccb69bbe8186084387f4fafd60d7ecbabc71 AS cli
ARG COPILOT_VERSION=1.0.105
ARG COPILOT_SHA256=aba1f06f27bc0447656647f76e0a4cf58b4b2a341c75ff04a750982b6aeb6bd2
ARG TARGETARCH
ADD --checksum=sha256:$COPILOT_SHA256 \
    https://github.com/github/copilot-cli/releases/download/v$COPILOT_VERSION/copilot-linux-x64.tar.gz \
    /tmp/copilot.tar.gz
RUN if [ "$TARGETARCH" != amd64 ]; then echo "The planner's image is for linux/amd64 alone." >&2; exit 1; fi \
    && mkdir /out && tar -xzf /tmp/copilot.tar.gz -C /out copilot

# The CLI unpacks its package on its first start, into the user's cache folder; unpacking it here,
# into nonroot's, lets it run on a read-only root file system and keeps its code read-only too.
FROM gcr.io/distroless/cc-debian13:nonroot@sha256:08476c244888bfb425bae1ae4dd7c17f093e972aab8426883d97f3c6f280f7c1 AS unpack
COPY --from=cli /out/copilot /usr/local/bin/copilot
USER 0
ENV COPILOT_SEA_EXTRACT_ONLY=1 COPILOT_PKG_CACHE_HOME=/home/nonroot/.cache/copilot
RUN ["/usr/local/bin/copilot"]

# The CLI needs glibc and the C++ runtime, which distroless's cc image has.
FROM gcr.io/distroless/cc-debian13:nonroot@sha256:08476c244888bfb425bae1ae4dd7c17f093e972aab8426883d97f3c6f280f7c1
COPY --from=cli /out/copilot /usr/local/bin/copilot
COPY --from=unpack /home/nonroot/.cache/copilot/pkg /home/nonroot/.cache/copilot/pkg
COPY --from=build /out/planner /usr/local/bin/planner
ENTRYPOINT ["/usr/local/bin/planner"]
