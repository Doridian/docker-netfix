package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

type DockerNetfixClient struct {
	client     *client.Client
	rootfsPath string
}

func NewDockerNetfixClient(rootfsPath string) (*DockerNetfixClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerNetfixClient{
		client:     cli,
		rootfsPath: rootfsPath,
	}, nil
}

func (c *DockerNetfixClient) CheckOnce(ctx context.Context) error {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		err = c.netfixCheck(ctx, container.ID, container.Names[0])
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *DockerNetfixClient) Listen(ctx context.Context) error {
	msgChan, errChan := c.client.Events(ctx, events.ListOptions{})

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

			err = c.netfixCheck(ctx, msg.Actor.ID, msg.Actor.Attributes["name"])
			if err != nil {
				return err
			}
		case err = <-errChan:
			return err
		}
	}
}

func (c *DockerNetfixClient) netfixCheck(ctx context.Context, containerID string, containerName string) error {
	log.Printf("Checking container %s", containerName)
	container, err := c.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}

	if container.State == nil {
		return nil
	}

	pid := container.State.Pid
	if pid < 1 {
		return nil
	}

	cmd := exec.Command("nsenter", fmt.Sprintf("--net=%s/proc/%d/ns/net", c.rootfsPath, pid), os.Args[0], "--netcheck", containerName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
