module github.com/zetetos/gt-telemetry

go 1.25.5

retract [v1.0.0, v1.5.1] // broken versions during migration to the zetetos org

require (
	github.com/dop251/goja v0.0.0-20251201205617-2bb4c724c0f9
	github.com/fatih/color v1.18.0
	github.com/gocarina/gocsv v0.0.0-20240520201108-78e41c74b4b1
	github.com/kaitai-io/kaitai_struct_go_runtime v0.10.0
	github.com/rs/zerolog v1.34.0
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.45.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
