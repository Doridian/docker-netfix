package main

import "context"

func main() {
	client, err := NewDockerNetfixClient()
	if err != nil {
		panic(err)
	}
	err = client.Listen(context.Background())
	if err != nil {
		panic(err)
	}
}
