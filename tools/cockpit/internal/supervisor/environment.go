package supervisor

import (
	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
)

// environment returns, by service name, the variables each service gets on top of the
// cockpit's environment: its listen address, and those of the services it depends on, its
// peers.
func environment(services []manifest.Service) map[string][]string {
	listens := make(map[string]string, len(services))
	for _, svc := range services {
		listens[svc.Name] = svc.Listen
	}

	vars := make(map[string][]string, len(services))

	for _, svc := range services {
		peers := make([]env.Peer, 0, len(svc.DependsOn))
		for _, name := range svc.DependsOn {
			peers = append(peers, env.Peer{Name: name, Listen: listens[name]})
		}

		vars[svc.Name] = env.Vars(svc.Listen, peers)
	}

	return vars
}
