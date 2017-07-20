FROM scratch

COPY kube-warden .

USER 65534:65534
ENTRYPOINT ["/kube-warden"]
