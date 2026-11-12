//go:build js && wasm

package portable

import "errors"

// OpenEmbedded is unavailable in the browser, where assets are fetched by
// JavaScript and registered with assetfs instead.
func OpenEmbedded() (*Engine, error) {
	return nil, errors.New("embedded assets are not available in the browser; register fetched assets instead")
}
