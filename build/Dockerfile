FROM librespace/kaitai:latest as builder
WORKDIR /src
RUN mkdir -p /out/gttelemetry
RUN --mount=type=bind,target=.,readwrite \
    kaitai-struct-compiler --target go --go-package gttelemetry --outdir /out internal/kaitai/gran_turismo_telemetry.ksy

FROM scratch AS output
COPY --from=builder /out/gttelemetry/gran_turismo_telemetry.go /