package stanclient

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/stan.go"
	"github.com/stretchr/testify/require"
)

type msgData struct {
	Data string `json:"data"`
}

type eventClientSuite struct {
	Client         *Client
	TestSubscriber testSubscriber
}

type testSubscriber struct {
	EvaluationData string
	name           string
}

func (t *testSubscriber) Subject() string {
	return "eventclient-test-subject"
}

func (t *testSubscriber) DurableName() string {
	return ""
}

func (t *testSubscriber) Name() string {
	return t.name
}

func (t *testSubscriber) MsgHandler() stan.MsgHandler {
	return func(m *stan.Msg) {
		data := msgData{}
		err := json.Unmarshal(m.Data, &data)
		if err != nil {
			fmt.Println(err)
			return
		}
		t.EvaluationData = data.Data
	}
}

func TestEventClient(t *testing.T) {
	s := &eventClientSuite{}
	s.setup(t)
	defer s.teardown(t)

	t.Run("testSubscribe", s.testSubscribe)
	t.Run("testUnsubscribe", s.testUnsubscribe)
	t.Run("testUnsubscribeAll", s.testUnsubscribeAll)
	t.Run("testUnsubscribeList", s.testUnsubscribeList)
}

func (s *eventClientSuite) testSubscribe(t *testing.T) {
	ts := &testSubscriber{
		name: "eventclient-test-subscribe",
	}
	err := s.Client.Subscribe(ts)
	require.Nil(t, err)

	sub, ok := s.Client.subscriptions[ts.Subject()+"-"+ts.Name()]
	require.True(t, ok)
	require.NotNil(t, sub)

	require.Empty(t, ts.EvaluationData)

	dataToPublish, err := json.Marshal(msgData{Data: "this_is_so_much_data"})
	require.Nil(t, err)

	err = s.Client.conn.Publish(ts.Subject(), dataToPublish)
	require.Nil(t, err)

	waitUntilNoPending(sub)

	require.Equal(t, "this_is_so_much_data", ts.EvaluationData)
}

func (s *eventClientSuite) testUnsubscribe(t *testing.T) {
	ts := &testSubscriber{
		name: "eventclient-test-unsubscribe",
	}
	err := s.Client.Subscribe(ts)
	require.Nil(t, err)

	sub, ok := s.Client.subscriptions[ts.Subject()+"-"+ts.Name()]
	require.True(t, ok)
	require.NotNil(t, sub)

	err = s.Client.Unsubscribe(ts.Subject() + "-" + ts.Name())
	require.Nil(t, err)

	sub, ok = s.Client.subscriptions[ts.Subject()+"-"+ts.Name()]
	require.True(t, ok)
	require.Nil(t, sub)
}

func (s *eventClientSuite) testUnsubscribeAll(t *testing.T) {
	ts := &testSubscriber{
		name: "eventclient-test-unsubscribe-all",
	}
	err := s.Client.Subscribe(ts)
	require.Nil(t, err)

	ts2 := &testSubscriber{
		name: "eventclient-test-unsubscribe-all-2",
	}
	err = s.Client.Subscribe(ts2)
	require.Nil(t, err)

	sub, ok := s.Client.subscriptions[ts.Subject()+"-"+ts.Name()]
	require.True(t, ok)
	require.NotNil(t, sub)

	sub2, ok := s.Client.subscriptions[ts2.Subject()+"-"+ts2.Name()]
	require.True(t, ok)
	require.NotNil(t, sub2)

	err = s.Client.Unsubscribe("all")
	require.Nil(t, err)

	sub, ok = s.Client.subscriptions[ts.Subject()+"-"+ts.Name()]
	require.True(t, ok)
	require.Nil(t, sub)

	sub, ok = s.Client.subscriptions[ts2.Subject()+"-"+ts2.Name()]
	require.True(t, ok)
	require.Nil(t, sub)
}

func (s *eventClientSuite) testUnsubscribeList(t *testing.T) {
	ts := &testSubscriber{
		name: "eventclient-test-unsubscribe-list",
	}
	err := s.Client.Subscribe(ts)
	require.Nil(t, err)

	ts2 := &testSubscriber{
		name: "eventclient-test-unsubscribe-list-2",
	}
	err = s.Client.Subscribe(ts2)
	require.Nil(t, err)

	require.Equal(t, 2, len(s.Client.Subscriptions()))

	err = s.Client.Unsubscribe(ts.Subject() + "-" + ts.Name())
	require.Nil(t, err)

	require.Equal(t, 1, len(s.Client.Subscriptions()))

	err = s.Client.Unsubscribe(ts2.Subject() + "-" + ts2.Name())
	require.Nil(t, err)

	require.Equal(t, 0, len(s.Client.Subscriptions()))
}

func (s *eventClientSuite) setup(t *testing.T) {
	var err error
	s.Client, err = New(Config{
		Enabled: true,
		ConnectRetry: Retry{
			Attempts: 5,
			Delay:    1,
		},
		ClientID:         "eventclient-test",
		ClusterID:        "test-cluster",
		NatsStreamingURL: "nats://localhost:4222",
	}, nil, false, nil)
	require.Nil(t, err)
}

func (s *eventClientSuite) teardown(t *testing.T) {
	s.Client.Close()
}

// waitUntilNoPending will block until the number of pending messages in the subscription is 0
// or timesout (200 milliseconds).
func waitUntilNoPending(sub stan.Subscription) {
	for i := 0; i < 20; i++ {
		pending, _, _ := sub.Pending()
		if pending == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
