module github.com/zetetos/gt-telemetry

go 1.24.4

retract [v1.0.0, v1.5.1] // broken versions during migration to the zetetos org

require (
	github.com/kaitai-io/kaitai_struct_go_runtime v0.11.0
	github.com/rs/zerolog v1.34.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.41.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
