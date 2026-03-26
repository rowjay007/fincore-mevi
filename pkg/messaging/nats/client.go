package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Subscriber interface {
	Subscribe(subject string, queue string, handler func(msg *nats.Msg)) error
}

type Client struct {
	nc *nats.Conn
}

func NewClient(url string) (*Client, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &Client{nc: nc}, nil
}

func (c *Client) Subscribe(subject string, queue string, handler func(msg *nats.Msg)) error {
	_, err := c.nc.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return fmt.Errorf("nats queue subscribe: %w", err)
	}
	return nil
}

func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}
