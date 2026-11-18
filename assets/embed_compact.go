//go:build aoikane_compact

package assets

import _ "embed"

//go:embed compact/models.tar.zst
var Bundle []byte

//go:embed compact/models.sha256
var BundleID string

//go:embed compact/openzl.wasm
var OpenZL []byte
