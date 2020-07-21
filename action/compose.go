package action

import "context"

// Compose is a docker-compose project
type Compose struct {
	path    string
	service string
	env     map[string]string
}

// Action run the project
func (c *Compose) Action(ctx context.Context) error {
	return nil
}
