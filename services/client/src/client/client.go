package client

import (
	"bufio"
	"net"
	"os"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/communication"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200
const END_MESSAGE = "END"

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
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
	defer client.conn.Close()

	inputFile, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}

	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)

	for scanner.Scan() {
		line := scanner.Text()
		bet := communication.CreateBetMessage(line, client.config.AgencyId)

		messageArgs := []any{"agency-id", client.config.AgencyId, "message", line}
		logger.Info("send-bet", logger.InProgress, messageArgs...)

		if err := communication.SendBetMessage(client.conn, bet); err != nil {
			logger.Error("send-bet", logger.Fail, messageArgs...)
			return err
		}

		logger.Info("send-bet", logger.Success, messageArgs...)

	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan-input-file", logger.Fail, "err", err)
		return err
	}

	if err := communication.SendBetMessage(client.conn, END_MESSAGE); err != nil {
		logger.Error("send-end", logger.Fail)
		return err
	}

	outputFile, err := os.Create(client.config.OutputFile)
	if err != nil {
		logger.Error("create-output-file", logger.Fail, "err", err)
		return err
	}

	defer outputFile.Close()

	for {
		messageArgs := []any{"agency-id", client.config.AgencyId}
		logger.Info("recv-winner", logger.InProgress, messageArgs...)

		responseWinner, err := communication.RecvWinnerMessage(client.conn)
		if err != nil {
			logger.Error("recv-winner", logger.Fail, messageArgs...)
			return err
		}

		if responseWinner == END_MESSAGE {
			break
		}

		if _, err := outputFile.WriteString(responseWinner + "\n"); err != nil {
			logger.Error("write-output-file", logger.Fail, messageArgs...)
			return err
		}

		logger.Info("recv-winner", logger.Success, messageArgs...)
	}

	logger.Info("test-file-send-to-server", logger.Success, "agency-id", client.config.AgencyId)

	return nil
}
