FROM scratch

COPY bin/kube-warden .

USER 65534:65534
ENTRYPOINT ["/kube-warden"]
