FROM golang:alpine as builder
RUN mkdir /build 
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor  -a -installsuffix cgo -ldflags '-extldflags "-static"' -o deployment .
FROM scratch
FROM microsoft/azure-cli
COPY --from=builder /build/deployment /app/
COPY ./azure_smc_template.json /app/
COPY ./scim_template.json /app/
WORKDIR /app
CMD ["/bin/bash"]
