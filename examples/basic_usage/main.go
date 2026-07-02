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

	// List the first page of devices in the token's team.
	devices, err := client.ListDevices(ctx, 1, 20)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("page %d of %d\n", devices.Current, devices.Last)
	for _, d := range devices.Items {
		if d == nil {
			continue
		}
		fmt.Printf("  %s\t%s\t(space %s)\n", d.ID, d.Name, d.SpaceID)
	}
}
