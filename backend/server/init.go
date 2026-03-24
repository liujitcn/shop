package server

import (
	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	NewHttpMiddleware,
	NewHttpServer,
	NewGrpcMiddleware,
	NewGRPCServer,
)
