module github.com/evstack/ev-node/compression

go 1.21

require (
	github.com/evstack/ev-node/core v0.0.0-00010101000000-000000000000
	github.com/klauspost/compress v1.17.4
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/evstack/ev-node/core => ../core