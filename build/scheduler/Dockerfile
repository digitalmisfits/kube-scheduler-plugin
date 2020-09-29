FROM golang:1.13.0

WORKDIR /tmp/scheduler
COPY . .

FROM alpine:3.12

COPY --from=0 /tmp/scheduler/bin/kube-scheduler /bin/kube-scheduler
COPY --from=0 /tmp/scheduler/etc/kubernetes/kube-scheduler.config.yaml /etc/kubernetes/kube-scheduler.config.yaml

WORKDIR /bin
CMD ["kube-scheduler"]