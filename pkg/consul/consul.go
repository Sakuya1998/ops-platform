package consul

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/consul/api"
)

type Registration struct {
	ID      string
	Name    string
	Address string
	Port    int
	Tags    []string
	Check   *api.AgentServiceCheck
}

type Client struct{ client *api.Client }

func NewClient(address string) (*Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = address
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create consul client: %w", err)
	}
	log.Printf("[Consul] Client initialized for: %s", address)
	return &Client{client: client}, nil
}

func (c *Client) Register(reg Registration) error {
	service := &api.AgentServiceRegistration{
		ID: reg.ID, Name: reg.Name, Address: reg.Address,
		Port: reg.Port, Tags: reg.Tags,
	}
	if reg.Check != nil {
		service.Check = reg.Check
	} else {
		service.Check = &api.AgentServiceCheck{
			HTTP: fmt.Sprintf("http://%s:%d/health", reg.Address, reg.Port),
			Interval: "10s", Timeout: "5s",
		}
	}
	if err := c.client.Agent().ServiceRegister(service); err != nil {
		return fmt.Errorf("register service: %w", err)
	}
	log.Printf("[Consul] Service registered: %s/%s on %s:%d", reg.Name, reg.ID, reg.Address, reg.Port)
	return nil
}

func (c *Client) Deregister(serviceID string) error {
	return c.client.Agent().ServiceDeregister(serviceID)
}

func (c *Client) Discover(serviceName string) ([]*api.ServiceEntry, error) {
	services, _, err := c.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("discover service: %w", err)
	}
	return services, nil
}

func (c *Client) WatchService(serviceName string, callback func([]*api.ServiceEntry)) {
	go func() {
		for {
			services, err := c.Discover(serviceName)
			if err == nil { callback(services) }
			time.Sleep(30 * time.Second)
		}
	}()
}
