FROM golang:1.15.0

WORKDIR /go/src/sigs.k8s.io/scheduler-plugins
COPY . .
RUN make build-scheduler

FROM alpine:3.12

COPY --from=0 /go/src/sigs.k8s.io/scheduler-plugins/bin/kube-scheduler /bin/kube-scheduler

WORKDIR /bin
CMD ["kube-scheduler"]