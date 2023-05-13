package main

import (
	"context"
	"flag"
)

func main() {
	netcheckPtr := flag.String("netcheck", "", "Internal")
	rootfsPath := flag.String("rootfs", "/", "RootFS")
	flag.Parse()

	if *netcheckPtr != "" {
		err := netcheck(*netcheckPtr)
		if err != nil {
			panic(err)
		}
		return
	}

	client, err := NewDockerNetfixClient(*rootfsPath)
	if err != nil {
		panic(err)
	}
	err = client.Listen(context.Background())
	if err != nil {
		panic(err)
	}
}
