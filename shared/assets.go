package shared

import _ "embed"

//go:embed diagram.schema.json
var DiagramSchema []byte

//go:embed library.json
var Library []byte
