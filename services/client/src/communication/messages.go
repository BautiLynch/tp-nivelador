package communication

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const HEADER_SIZE = 4
const BATCH_SEPARATOR = "\n"
const ACK_MESSAGE = "ACK"
const NACK_MESSAGE = "NACK"
const END_MESSAGE = "END"

func SendMessage(socket io.Writer, payload string) error {
	message := NewMessage()
	message = append(message, payload...)
	serializeMessage(message)

	return safe_socket.SendAll(socket, message)
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

func NewMessage() []byte {
	return make([]byte, HEADER_SIZE)
}

func ReuseMessage(message []byte) []byte {
	return message[:HEADER_SIZE]
}

func serializeMessage(message []byte) {
	payloadSize := len(message) - HEADER_SIZE
	binary.BigEndian.PutUint32(message[:HEADER_SIZE], uint32(payloadSize))
}

func SendBatchBetMessage(socket net.Conn, batch []byte) (bool, error) {
	serializeMessage(batch)

	if err := safe_socket.SendAll(socket, batch); err != nil {
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
