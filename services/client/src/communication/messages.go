package communication

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const HEADER_SIZE = 4
const BATCH_SEPARATOR = "\n"
const ACK_MESSAGE = "ACK"
const NACK_MESSAGE = "NACK"
const END_MESSAGE = "END"

func SendMessage(socket io.Writer, payload string) error {

	payloadBytes := []byte(payload)

	header := CreateHeader(payloadBytes)

	if err := safe_socket.SendAll(socket, header); err != nil {
		return err
	}

	if err := safe_socket.SendAll(socket, payloadBytes); err != nil {
		return err
	}

	return nil
}

func RecvMessage(socket io.Reader) (string, bool, error) {
	header, err := safe_socket.RecvAll(socket, HEADER_SIZE)

	if err != nil {
		return "", true, err
	}

	size := int(binary.BigEndian.Uint32(header))

	winnerMessage, err := safe_socket.RecvAll(socket, size)

	if err != nil {
		return "", true, err
	}

	winner := string(winnerMessage)

	if winner == END_MESSAGE {
		return "", true, nil
	}

	return string(winner), false, nil
}

func CreateBetMessage(line string, agencyId string) string {
	bet := []string{agencyId, line}
	return strings.Join(bet, ",")
}

func CreateHeader(payloadBytes []byte) []byte {
	header := make([]byte, HEADER_SIZE)
	binary.BigEndian.PutUint32(header, uint32(len(payloadBytes)))
	return header
}

func SendBatchBetMessage(socket net.Conn, batch []string) (bool, error) {
	payload := strings.Join(batch, BATCH_SEPARATOR)
	payloadBytes := []byte(payload)

	header := CreateHeader(payloadBytes)

	if err := safe_socket.SendAll(socket, header); err != nil {
		return false, err
	}

	if err := safe_socket.SendAll(socket, payloadBytes); err != nil {
		return false, err
	}

	ackPayload, finished, err := RecvMessage(socket)
	if err != nil {
		return false, err
	}

	if finished {
		return false, fmt.Errorf("unexpected end message while waiting for batch ack")
	}

	if ackPayload == NACK_MESSAGE {
		return false, nil
	}

	if ackPayload == ACK_MESSAGE {
		return true, nil
	}

	return false, fmt.Errorf(
		"unexpected batch ack: %q",
		ackPayload,
	)
}

func SendBetsFinished(socket io.Writer) error {
	if err := SendMessage(socket, END_MESSAGE); err != nil {
		logger.Error("send-end", logger.Fail)
		return err
	}

	return nil
}
