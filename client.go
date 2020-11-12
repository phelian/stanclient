package stanclient

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/nats-io/stan.go"
)

var errNotEnabled = errors.New("Client not configued to be enabled")

// Client wrapper for stan connection
type Client struct {
	lock          *sync.Mutex
	conn          stan.Conn
	subscriptions map[string]stan.Subscription
	Logger        ClientLogger
	reconnectFunc func() error
	clientID      string
	Config
}

// New returns a connected eventclient or error
func New(config Config, logger ClientLogger, wrapID bool, reconnectFunc func() error) (*Client, error) {
	lgr := logger
	if logger == nil {
		lgr = &EmptyLogger{}
	}

	clientID := config.ClientID
	if wrapID {
		clientID = wrapClientID(clientID)
	}
	client := &Client{
		Config:        config,
		subscriptions: make(map[string]stan.Subscription),
		Logger:        lgr,
		clientID:      clientID,
		reconnectFunc: reconnectFunc,
		lock:          &sync.Mutex{},
	}
	if config.Enabled {
		if err := client.connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to %s %w", config.NatsStreamingURL, err)
		}
	}

	return client, nil
}

// connect to nats-streaming via stan
func (c *Client) connect() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.conn != nil {
		return nil
	}

	if !c.Enabled {
		return errNotEnabled
	}

	err := retry.Do(
		func() error {
			var retryErr error
			c.conn, retryErr = stan.Connect(c.ClusterID, c.clientID, stan.NatsURL(c.NatsStreamingURL),
				stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
					c.Logger.Info("Connection lost to nats-streaming server")
					if c.reconnectFunc == nil {
						c.Logger.Fatal(fmt.Sprintf("Connection lost, reason: %v", reason))
					}

					c.conn = nil
					err := retry.Do(
						func() error {
							if err := c.connect(); err != nil {
								return err
							}

							c.Logger.Info("Successfully reconnected to nats-streaming server")
							return nil
						},
						retry.OnRetry(func(n uint, err error) {
							c.Logger.Info(fmt.Sprintf("Reconnection try #%d failed with: %s", n, err))
						}),
						retry.Delay(time.Duration(c.ReconnectRetry.Delay)*time.Second),
						retry.DelayType(retry.FixedDelay),
						retry.Attempts(c.ReconnectRetry.Attempts),
					)

					if err != nil {
						c.Logger.Fatal(fmt.Sprintf("All attempts to reconnect to the streaming server failed %s", err.Error()))
					}

					if err := c.reconnectFunc(); err != nil {
						c.Logger.Fatal(fmt.Sprintf("Reconnection func failed %s", err.Error()))
					}
				}))

			if retryErr != nil {
				return fmt.Errorf("can't connect make sure a NATS Streaming Server is running at: %s %w", c.NatsStreamingURL, retryErr)
			}

			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			c.Logger.Info(fmt.Sprintf("Connect() retry failed with retry-number: %d %s", n, err.Error()))
		}),
		retry.Delay(time.Duration(c.ConnectRetry.Delay)*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(c.ConnectRetry.Attempts),
	)

	if err != nil {
		return fmt.Errorf("all retries failed to connect to %s: %w", c.NatsStreamingURL, err)
	}
	c.Logger.Info(fmt.Sprintf("Connected to %s clusterID: [%s] clientID: [%s]", c.NatsStreamingURL, c.ClusterID, c.clientID))
	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Subscribe an incoming subscriber to clients connection
func (c *Client) Subscribe(subscriber Subscriber, opts ...stan.SubscriptionOption) error {
	if err := c.connect(); err != nil {
		if errors.Is(err, errNotEnabled) {
			return nil
		}
		return fmt.Errorf("failed to connect %w", err)
	}

	s, err := c.conn.Subscribe(
		subscriber.Subject(),
		subscriber.MsgHandler(),
		opts...,
	)
	if err != nil {
		return fmt.Errorf("error subscribing to '%s' on cluster '%s': %w", subscriber.Subject(), c.ClusterID, err)
	}
	c.subscriptions[subscriber.Subject()+"-"+subscriber.Name()] = s
	return nil
}

// QueueSubscribe an incoming subscriber to clients connection
func (c *Client) QueueSubscribe(subscriber Subscriber, queueGroup string, opts ...stan.SubscriptionOption) error {
	if err := c.connect(); err != nil {
		if errors.Is(err, errNotEnabled) {
			return nil
		}
		return fmt.Errorf("failed to connect %w", err)
	}

	s, err := c.conn.QueueSubscribe(
		subscriber.Subject(),
		queueGroup,
		subscriber.MsgHandler(),
		opts...,
	)
	if err != nil {
		return fmt.Errorf("error queue subscribing to '%s' on cluster '%s': %w", subscriber.Subject(), c.ClusterID, err)
	}
	c.subscriptions[subscriber.Subject()+"-"+queueGroup+"-"+subscriber.Name()] = s
	return nil
}

// Unsubscribe <subscriber> or special case <all> for all subscriptions. Will leave the map key with nil value for unsubscribed.
func (c *Client) Unsubscribe(subscriber string) error {
	if err := c.connect(); err != nil {
		if errors.Is(err, errNotEnabled) {
			return nil
		}
		return fmt.Errorf("failed to reconnect %w", err)
	}

	switch subscriber {
	case "all":
		for subscriber, subscription := range c.subscriptions {
			if subscription == nil {
				// already unsubscribed
				continue
			}
			if err := subscription.Unsubscribe(); err != nil {
				return fmt.Errorf("unsubscribe all failed %w", err)
			}
			c.subscriptions[subscriber] = nil
			c.Logger.Info(fmt.Sprintf("successfully unsubscribed subscriber %s", subscriber))
		}
	default:
		sub, ok := c.subscriptions[subscriber]
		if !ok || sub == nil {
			return fmt.Errorf("could not find subscription %s amongst the current subscriptions", subscriber)
		}
		if err := sub.Unsubscribe(); err != nil {
			return fmt.Errorf("unsubscribe %s failed %w", subscriber, err)
		}
		c.subscriptions[subscriber] = nil

		c.Logger.Info(fmt.Sprintf("successfully unsubscribed subscriber %s", subscriber))
	}

	return nil
}

// Subscriptions return a list of current subscribtions
func (c *Client) Subscriptions() []string {
	result := []string{}
	for subscription, subscriber := range c.subscriptions {
		if subscriber != nil {
			result = append(result, subscription)
		}
	}
	return result
}

// wrapClientID prefixes the provided client id with the host name, to make it unique.
func wrapClientID(clientID string) string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "<error>"
	}

	// Replace all non-alphanumeric chars with something benign that will work in the client id.
	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	hostname = re.ReplaceAllString(hostname, "-")

	// Special case for running locally, add random
	if runtime.GOOS == "darwin" {
		hostname += strconv.Itoa(rand.Intn(100))
		return fmt.Sprintf("%s-%s", clientID, hostname)
	}
	return fmt.Sprintf("%s-%s", clientID, hostname)
}
