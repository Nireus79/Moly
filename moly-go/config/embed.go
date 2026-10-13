package config

import "embed"

// Files holds the configuration files shipped inside the binary, so an installed Moly needs no config folder next to it.
//
//go:embed *.yaml
var Files embed.FS
