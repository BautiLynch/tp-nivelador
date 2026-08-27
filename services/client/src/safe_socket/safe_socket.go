package safe_socket

import "io"

//TODO: Complete with a short-read/short-write tolerant implementation

func SendAll(socket io.Writer, bytes []byte) error {
	wroteBytes := 0
	for wroteBytes < len(bytes) {
		n, err := socket.Write(bytes[wroteBytes:])
		if err != nil {
			return err
		}
		wroteBytes += n
	}
	return nil
}

func RecvAll(socket io.Reader, size int) ([]byte, error) {
	buff := make([]byte, size)
	readBytes := 0
	for readBytes < size {
		n, err := socket.Read(buff[readBytes:])
		if err != nil {
			return nil, err
		}
		readBytes += n
	}
	return buff, nil
}
