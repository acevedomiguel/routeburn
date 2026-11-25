package routeburn_test

import (
	"context"
	"fmt"

	routeburn "github.com/acevedomiguel/routeburn"
)

func ExampleFetch() {
	snap, err := routeburn.Fetch(context.Background(), fakeAPI{}, routeburn.FetchOptions{Now: fixedTime()})
	if err != nil {
		panic(err)
	}
	fmt.Println(snap.Account.Remaining)
	// Output: 42.1
}
