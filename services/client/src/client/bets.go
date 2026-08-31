package client

import (
	"bufio"
	"os"
	"strings"

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
	bets := make([]string, 0, client.config.BatchSize)
	batchNumber := 1
	for scanner.Scan() {
		line := scanner.Text()
		bet := CreateBetMessage(line, client.config.AgencyId)
		bets = append(bets, bet)

		if len(bets) < client.config.BatchSize {
			continue
		}

		messageArgs := []any{"agency-id", client.config.AgencyId, "message", batchNumber}

		accepted, err := communication.SendBatchBetMessage(client.conn, bets)
		if err != nil {
			logger.Error("send-batch", logger.Fail, messageArgs...)
			return err
		}

		if !accepted {
			logger.Warn("send-batch", logger.Fail, messageArgs...)
		}

		bets = bets[:0]
		batchNumber += 1
	}

	if len(bets) != 0 {
		messageArgs := []any{"agency-id", client.config.AgencyId, "message", batchNumber}
		logger.Info("send-batch", logger.InProgress, messageArgs...)

		accepted, err := communication.SendBatchBetMessage(client.conn, bets)
		if err != nil {
			logger.Error("send-batch", logger.Fail, messageArgs...)
			return err
		}
		if !accepted {
			logger.Warn("send-batch", logger.Fail, messageArgs...)
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

	for {
		messageArgs := []any{"agency-id", client.config.AgencyId}
		logger.Info("recv-winner", logger.InProgress, messageArgs...)

		responseWinner, finished, err := communication.RecvMessage(client.conn)
		if err != nil {
			logger.Error("recv-winner", logger.Fail, messageArgs...)
			return err
		}

		if finished {
			break
		}

		if _, err := outputFile.WriteString(responseWinner + "\n"); err != nil {
			logger.Error("write-output-file", logger.Fail, messageArgs...)
			return err
		}

		logger.Info("recv-winner", logger.Success, messageArgs...)
	}

	return nil
}

func CreateBetMessage(line string, agencyId string) string {
	bet := []string{agencyId, line}
	return strings.Join(bet, ",")
}
