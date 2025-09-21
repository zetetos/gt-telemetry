FROM librespace/kaitai:latest as builder
WORKDIR /src
RUN mkdir -p /out/gttelemetry
RUN --mount=type=bind,target=.,readwrite \
    kaitai-struct-compiler --target go --go-package telemetry --outdir /out internal/kaitai/gran_turismo_telemetry.ksy

FROM scratch AS output
COPY --from=builder /out/telemetry/gran_turismo_telemetry.go /telemetry.go