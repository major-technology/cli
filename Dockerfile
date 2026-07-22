# Compile-job image: runs `major project compile-and-report` as a K8s Job.
FROM golang:1.24.2-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build \
	-ldflags "-X 'github.com/major-technology/cli/cmd.Version=${VERSION}' -X 'github.com/major-technology/cli/cmd.configFile=configs/prod.json'" \
	-o /major .

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates
COPY --from=build /major /usr/local/bin/major
ENTRYPOINT ["major", "project", "compile-and-report"]
