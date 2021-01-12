package compose

import (
	"fmt"
	"sort"
)

// NewServiceGraph generates a graph of deps from a compose description
func (c Compose) NewServiceGraph() ServiceGraph {
	// init graph
	graph := make(ServiceGraph)

	// range over all services and populate the graph
	for service, value := range c.Services {
		data, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		deps, ok := data["depends_on"].([]interface{})
		if !ok {
			continue
		}

		for _, value := range deps {
			dep, ok := value.(string)
			if !ok {
				continue
			}
			graph[service] = append(graph[service], dep)
		}

	}

	return graph
}

// ServiceDepth represents the level of deps for a services
type ServiceDepth map[string]int

func (sd ServiceDepth) findLeader() (string, error) {

	if len(sd) == 0 {
		return "", fmt.Errorf("Empty graph")
	}

	// a tmp structure used to order the input map
	type smap struct {
		Key   string
		Value int
	}

	// a tmp value used for ordering
	var tmp []smap

	for k, v := range sd {
		tmp = append(tmp, smap{k, v})
	}

	// using sort.Slice from go 1.8
	sort.Slice(tmp, func(a, b int) bool {
		return tmp[a].Value > tmp[b].Value
	})

	switch {
	case len(tmp) == 1:
		return tmp[0].Key, nil
	case len(tmp) > 1:
		if tmp[0].Value == tmp[1].Value {
			// ensure a sorted response to get similar error message between two calls
			ambiguity := []string{tmp[0].Key, tmp[1].Key}
			sort.Strings(ambiguity)
			return "", fmt.Errorf("Leader ambiguity between nodes %s and %s", ambiguity[0], ambiguity[1])
		}

		return tmp[0].Key, nil
	default:
		return "", fmt.Errorf("Unexpected case in graph structure")
	}
}

// Len is used by the sort interface
func (sd ServiceDepth) Len() int {
	return len(sd)
}

// ServiceGraph represents a map of services to dependencies
type ServiceGraph map[string]([]string)

// ByServiceDepth computes deps depth by service
func (s ServiceGraph) ByServiceDepth() ServiceDepth {

	d := make(ServiceDepth)

	for k := range s {
		d[k] = s.serviceDepth(k, d)
	}

	return d

}

func (s ServiceGraph) serviceDepth(index string, memory ServiceDepth) int {

	if _, found := s[index]; !found {
		return 1
	}

	if depth, ok := memory[index]; ok {
		return depth
	}

	var childs int
	for _, child := range s[index] {
		childs += s.serviceDepth(child, memory)
	}

	return childs
}
