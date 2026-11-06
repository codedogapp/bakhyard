# Build from the repo's root: docker build -f services/web/Dockerfile . (docs/plan.md, decision 138)
FROM --platform=$BUILDPLATFORM node:26.13.0-alpine@sha256:c6ad9e9c1d80f6eb607c26b4b0a132ff7a2344fad1c78ea3baa2a7d4c8de6d94 AS app
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

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS build
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

FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
COPY --from=build /out/web /usr/local/bin/web
ENTRYPOINT ["/usr/local/bin/web"]
