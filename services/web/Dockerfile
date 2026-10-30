# Build from the repo's root: docker build -f services/web/Dockerfile . (docs/plan.md, decision 138)
FROM --platform=$BUILDPLATFORM node:26.12.1-alpine@sha256:40cd011b55415bc894e45b02fdd5a5dfde056921b7d23a64c74f80497af48797 AS app
# renovate: datasource=npm depName=pnpm
ARG PNPM_VERSION=12.16.1
RUN npm install --global --ignore-scripts "pnpm@$PNPM_VERSION"
WORKDIR /src/apps/web
COPY apps/web/package.json apps/web/pnpm-lock.yaml apps/web/pnpm-workspace.yaml ./
# The lockfile links deepmerge to the app's own merge (decision 148).
COPY apps/web/overrides/ overrides/
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY apps/web/ ./
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY internal/ internal/
COPY services/web/ services/web/
COPY --from=app /src/services/web/dist/app/ services/web/dist/app/
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/web ./services/web

FROM gcr.io/distroless/static-debian13:nonroot@sha256:18f4cc26c9ffa7f20a359298be7deadac5c607c8e03a02b8119f1a5fc2f82c99
COPY --from=build /out/web /usr/local/bin/web
ENTRYPOINT ["/usr/local/bin/web"]
