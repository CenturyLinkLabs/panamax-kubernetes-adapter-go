package main // import "github.com/CenturyLinkLabs/panamax-kubernetes-adapter-go"

import (
	"github.com/CenturyLinkLabs/panamax-kubernetes-adapter-go/adapter"
	"github.com/CenturyLinkLabs/pmxadapter"
)

func main() {
	adapter := adapter.KubernetesAdapter{}
	server := pmxadapter.NewServer(adapter)

	server.Start()
}
