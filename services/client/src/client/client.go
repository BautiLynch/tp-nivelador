package client

import (
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
	BatchSize  int
}

type Client struct {
	conn        net.Conn
	config      ClientConfig
	terminating atomic.Bool
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	client := &Client{conn: conn, config: config}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) Run() error {
	signalChannel := make(chan os.Signal, 1)
	terminated := make(chan struct{})
	handlerEnded := make(chan struct{})

	client.terminating.Store(false)

	signal.Notify(signalChannel, syscall.SIGTERM)
	go client.handleSignal(signalChannel, terminated, handlerEnded)

	defer client.terminateClient(signalChannel, terminated, handlerEnded)
	if err := SendBets(client); err != nil {
		if client.terminating.Load() {
			return nil
		}
		return err
	}

	if client.terminating.Load() {
		return nil
	}

	if err := GetWinners(client); err != nil {
		if client.terminating.Load() {
			return nil
		}
		return err
	}

	logger.Info("test-file-send-to-server", logger.Success, "agency-id", client.config.AgencyId)

	return nil
}

func (client *Client) handleSignal(signalChannel <-chan os.Signal, terminated <-chan struct{}, handlerEnded chan<- struct{}) {
	defer close(handlerEnded)

	select {
	case <-signalChannel:
		client.terminating.Store(true)
		client.conn.Close()

	case <-terminated:
	}
}

func (client *Client) terminateClient(signalChannel chan os.Signal, terminated chan struct{}, handlerEnded <-chan struct{}) {
	signal.Stop(signalChannel)
	close(terminated)
	client.conn.Close()
	<-handlerEnded
}
