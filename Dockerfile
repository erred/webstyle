FROM golang:alpine AS build

ENV CGO_ENABLED=0
WORKDIR /workspace
COPY . .
RUN go build -o /bin/webrender ./cmd/webrender

FROM scratch

COPY --from=build /bin/webrender /bin/
ENTRYPOINT [ "/bin/webrender" ]
