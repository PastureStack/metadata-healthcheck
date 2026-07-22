FROM golang:1.26.5-bookworm AS build

WORKDIR /src
COPY go.mod ./
COPY test/integration/fixture ./test/integration/fixture
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /fixture ./test/integration/fixture

FROM scratch
COPY --from=build /fixture /fixture
ENTRYPOINT ["/fixture"]
