// Example: basic usage of the operator management API client.
//
//	QUALITHM_API_TOKEN=qmt_... go run ./examples/basic_usage
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	operator "github.com/qualithm/operator-go"
)

// ---------------------------------------------------------------------------
// Entrypoint
// ---------------------------------------------------------------------------

func main() {
	token := os.Getenv("QUALITHM_API_TOKEN")
	if token == "" {
		log.Fatal("set QUALITHM_API_TOKEN to a member API token (qmt_...)")
	}

	// Point at a local server with QUALITHM_API_URL; defaults to production.
	opts := []operator.Option{}
	if base := os.Getenv("QUALITHM_API_URL"); base != "" {
		opts = append(opts, operator.WithBaseURL(base))
	}

	client, err := operator.New(token, opts...)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Walk every device in the token's team, a page at a time.
	fetch := func(ctx context.Context, cursor string) (operator.Page[operator.Device], error) {
		return client.ListDevices(ctx, cursor, operator.WalkLimit)
	}
	for d, err := range operator.All(ctx, fetch) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s\t%s\t(space %s)\n", d.ID, d.Name, d.SpaceID)
	}
}
