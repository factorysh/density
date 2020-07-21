package action

import "context"

// Compose is a docker-compose project
type Compose struct {
}

// Action run the project
func (c *Compose) Action(ctx context.Context) error {
	return nil
}
