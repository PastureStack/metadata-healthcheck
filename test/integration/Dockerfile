FROM golang:1.27.0-bookworm@sha256:484ef6066fa69acb059fdfeda7ba2b8f7391f2ef6abc6f9b8411e669ebd56466 AS build

WORKDIR /src
COPY go.mod ./
COPY test/integration/fixture ./test/integration/fixture
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /fixture ./test/integration/fixture

FROM scratch
COPY --from=build /fixture /fixture
ENTRYPOINT ["/fixture"]
