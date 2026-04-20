package gorse

import (
	"shop/api/gen/go/conf"

	client "github.com/gorse-io/gorse-go"
)

type Gorse struct {
	gorseClient *client.GorseClient
}

func NewGorse(cfg *conf.Gorse) *Gorse {
	gorseClient := client.NewGorseClient(cfg.GetEntryPoint(), cfg.GetApiKey())
	return &Gorse{
		gorseClient: gorseClient,
	}
}
