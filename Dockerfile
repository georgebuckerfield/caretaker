FROM scratch

COPY bin/caretaker .

USER 65534:65534
ENTRYPOINT ["/caretaker"]
