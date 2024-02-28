FROM scratch

COPY lesshero /

ENTRYPOINT ["/lesshero", "-c", "lesshero.html"]
