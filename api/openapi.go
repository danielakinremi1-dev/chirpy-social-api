// Package openapi embeds the OpenAPI spec and the Swagger UI page that renders it.
package openapi

import _ "embed"

//go:embed openapi.yaml
var Spec []byte

//go:embed docs.html
var DocsPage []byte
