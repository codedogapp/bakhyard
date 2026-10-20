package manifest

import (
	"fmt"
	"slices"
	"strings"
)

// startOrder puts each service after the services it depends on, and otherwise keeps the
// given order. It fails on a dependency cycle and names it. Every dependency must be a known
// service.
func startOrder(services []Service) ([]Service, error) {
	sorter := orderer{
		byName:  make(map[string]Service, len(services)),
		visited: make(map[string]bool, len(services)),
	}

	for _, svc := range services {
		sorter.byName[svc.Name] = svc
	}

	for _, svc := range services {
		err := sorter.visit(svc.Name)
		if err != nil {
			return nil, err
		}
	}

	return sorter.order, nil
}

// orderer walks the dependencies depth first, adding each service once all of its own are in.
type orderer struct {
	byName  map[string]Service
	visited map[string]bool
	// path is the chain of services being visited, so a service met again on it closes a cycle.
	path  []string
	order []Service
}

func (o *orderer) visit(name string) error {
	if o.visited[name] {
		return nil
	}

	if start := slices.Index(o.path, name); start >= 0 {
		cycle := slices.Concat(o.path[start:], []string{name})

		return fmt.Errorf("dependency cycle: %s", strings.Join(cycle, " → "))
	}

	o.path = append(o.path, name)
	svc := o.byName[name]

	for _, dependency := range svc.DependsOn {
		err := o.visit(dependency)
		if err != nil {
			return err
		}
	}

	o.path = o.path[:len(o.path)-1]
	o.visited[name] = true
	o.order = append(o.order, svc)

	return nil
}
