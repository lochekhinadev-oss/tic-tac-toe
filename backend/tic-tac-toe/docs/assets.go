package docs

import _ "embed"

//go:embed swagger.json
var OpenAPIJSON []byte

//go:embed swagger.yaml
var OpenAPIYAML []byte
