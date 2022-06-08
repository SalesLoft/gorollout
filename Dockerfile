ARG go_version=1.18
FROM golang:${go_version}-alpine as build

WORKDIR /src

# install dependencies
ARG GOPROXY=https://proxy.golang.org
COPY go.mod go.sum ./
RUN go mod download

COPY feature.go manager.go ./
COPY cmd ./cmd

RUN CGO_ENABLED=0 go build -o /rollout ./cmd/rollout/main.go

FROM scratch

COPY --from=build /rollout /rollout
CMD [ "/rollout" ]
