import socket


def recv_all(socket: socket.socket, size):
    bytesAmount = 0
    readBytes = []
    while bytesAmount < size:
        read = socket.recv(size - bytesAmount)
        if not read:
            return b""
        readBytes.append(read)
        bytesAmount += len(read)
    return b"".join(readBytes)


def send_all(socket: socket.socket, bytes):
    wroteBytes = 0
    while wroteBytes < len(bytes):
        n = socket.send(bytes[wroteBytes:])
        wroteBytes += n
    return wroteBytes
