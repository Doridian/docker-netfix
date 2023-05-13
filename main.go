package main

import (
	"context"
	"flag"
)

func main() {
	netcheckPtr := flag.Bool("netcheck", false, "Internal")
	rootfsPath := flag.String("rootfs", "/", "RootFS")
	flag.Parse()

	if *netcheckPtr {
		err := netcheck()
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
