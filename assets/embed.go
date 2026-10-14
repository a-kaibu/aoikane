// Package assets embeds the reconstructed engine's data: the analysis
// dictionary, rules and tables, the boundary and synthesis models and the
// chunker and dependency models. Neural inference uses Go only; no runtime library is embedded.
package assets

import "embed"

//go:embed boundmodel cabocha data dict dict-kansai rules synth
var FS embed.FS
