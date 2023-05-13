package main

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

type DockerNetfixClient struct {
	client *client.Client
}

func NewDockerNetfixClient() (*DockerNetfixClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerNetfixClient{
		client: cli,
	}, nil
}

func (c *DockerNetfixClient) CheckOnce(ctx context.Context) error {
	containers, err := c.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		c.netfixCheck(container.ID)
	}

	return nil
}

func (c *DockerNetfixClient) Listen(ctx context.Context) error {
	msgChan, errChan := c.client.Events(ctx, types.EventsOptions{})

	err := c.CheckOnce(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgChan:
			if msg.Type != events.ContainerEventType {
				continue
			}

			if msg.Action != "start" {
				continue
			}

			c.netfixCheck(msg.Actor.ID)
		case err = <-errChan:
			return err
		}
	}
}

func (c *DockerNetfixClient) netfixCheck(containerID string) {
	log.Printf("Checking container %s", containerID)
}
