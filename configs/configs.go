package configs

import (
	_ "embed"
)

//go:embed prod.json
var ProdConfig []byte

//go:embed local.json
var LocalConfig []byte
