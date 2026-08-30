package communication

import (
	"encoding/binary"
	"io"
	"strings"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const HEADER_SIZE = 4

func SendBetMessage(socket io.Writer, payload string) error {

	payloadBytes := []byte(payload)

	header := make([]byte, HEADER_SIZE)
	binary.BigEndian.PutUint32(header, uint32(len(payloadBytes)))

	if err := safe_socket.SendAll(socket, header); err != nil {
		return err
	}

	if err := safe_socket.SendAll(socket, payloadBytes); err != nil {
		return err
	}

	return nil
}

func RecvWinnerMessage(socket io.Reader) (string, error) {
	header, err := safe_socket.RecvAll(socket, HEADER_SIZE)

	if err != nil {
		return "", err
	}

	size := int(binary.BigEndian.Uint32(header))

	winner, err := safe_socket.RecvAll(socket, size)

	if err != nil {
		return "", err
	}

	return string(winner), nil
}

func CreateBetMessage(line string, agencyId string) string {
	bet := []string{agencyId, line}
	return strings.Join(bet, ",")
}
