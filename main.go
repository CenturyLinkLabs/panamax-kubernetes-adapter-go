package main // import "github.com/CenturyLinkLabs/panamax-kubernetes-adapter"

import (
	"github.com/CenturyLinkLabs/panamax-kubernetes-adapter/adapter"
	"github.com/CenturyLinkLabs/pmxadapter"
)

func main() {
	adapter := adapter.KubernetesAdapter{}
	server := pmxadapter.NewServer(adapter)

	server.Start()
}
