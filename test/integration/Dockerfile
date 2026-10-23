FROM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build

WORKDIR /src
COPY go.mod ./
COPY test/integration/fixture ./test/integration/fixture
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /fixture ./test/integration/fixture

FROM scratch
COPY --from=build /fixture /fixture
ENTRYPOINT ["/fixture"]
