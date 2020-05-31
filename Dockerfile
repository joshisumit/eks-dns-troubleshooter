FROM golang:alpine AS builder

WORKDIR /go/src/github.com/joshisumit/eks-dns-troubleshooter/

COPY . .

RUN make build

FROM amazonlinux:2

WORKDIR /app

RUN yum update -y && \
    yum install -y bind-utils && \
    yum clean all

COPY --from=builder /go/src/github.com/joshisumit/eks-dns-troubleshooter/eks-dnshooter /app/

ENTRYPOINT ["/app/eks-dnshooter"]