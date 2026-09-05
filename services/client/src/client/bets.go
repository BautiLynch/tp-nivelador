package client

import (
	"bufio"
	"io"
	"os"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/communication"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

func SendBets(client *Client) error {
	inputFile, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}

	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	batch := communication.NewMessage()
	betsNumber := 0
	batchNumber := 1
	for scanner.Scan() {
		if client.terminating.Load() {
			return nil
		}
		if betsNumber > 0 {
			batch = append(batch, communication.BATCH_SEPARATOR...)
		}
		batch = append(batch, client.config.AgencyId...)
		batch = append(batch, ',')
		batch = append(batch, scanner.Bytes()...)
		betsNumber += 1

		if betsNumber < client.config.BatchSize {
			continue
		}

		accepted, err := communication.SendBatchBetMessage(client.conn, batch)
		if err != nil {
			logger.Error("send-batch", logger.Fail, "agency-id", client.config.AgencyId, "message", batchNumber)
			return err
		}

		if !accepted {
			logger.Warn("send-batch", logger.Fail, "agency-id", client.config.AgencyId, "message", batchNumber)
		}

		batch = communication.ReuseMessage(batch)
		betsNumber = 0
		batchNumber += 1
	}

	if betsNumber != 0 {
		accepted, err := communication.SendBatchBetMessage(client.conn, batch)
		if err != nil {
			logger.Error("send-batch", logger.Fail, "agency-id", client.config.AgencyId, "message", batchNumber)
			return err
		}
		if !accepted {
			logger.Warn("send-batch", logger.Fail, "agency-id", client.config.AgencyId, "message", batchNumber)
		}

	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan-input-file", logger.Fail, "err", err)
		return err
	}

	if err := communication.SendBetsFinished(client.conn); err != nil {
		logger.Error("send-end", logger.Fail)
		return err
	}

	return nil
}

func GetWinners(client *Client) error {
	outputFile, err := os.Create(client.config.OutputFile)
	if err != nil {
		logger.Error("create-output-file", logger.Fail, "err", err)
		return err
	}

	defer outputFile.Close()

	messageArgs := []any{"agency-id", client.config.AgencyId}
	for {
		logger.Info("recv-winner", logger.InProgress, messageArgs...)

		responseWinner, finished, err := communication.RecvMessage(client.conn)
		if err != nil {
			logger.Error("recv-winner", logger.Fail, messageArgs...)
			return err
		}

		if finished {
			break
		}

		line := responseWinner + "\n"
		n, err := outputFile.WriteString(line)

		if err != nil {
			logger.Error("write-output-file", logger.Fail, messageArgs...)
			return err
		}

		if n != len(line) {
			logger.Error("write-output-file", logger.Fail, messageArgs...)
			return io.ErrShortWrite
		}

		logger.Info("recv-winner", logger.Success, messageArgs...)
	}

	return nil
}
