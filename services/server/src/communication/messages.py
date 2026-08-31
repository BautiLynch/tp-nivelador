import socket
from safe_socket import recv_all, send_all
from lottery import Bet, Lottery

HEADER_SIZE = 4
BET_ARGUMENTS = 6
END_MESSAGE = "END"
ACK_MESSAGE = "ACK"
NACK_MESSAGE = "NACK"

def send_message(socket: socket.socket, winner: str):
    bytes_winner = winner.encode()
    size = len(bytes_winner)
    header_bytes = size.to_bytes(HEADER_SIZE)
    send_all(socket, header_bytes)
    send_all(socket, bytes_winner)

def send_messages(socket: socket.socket, winners: list[Bet]):
    for winner in winners:
        bet_str = create_winner_message(winner)
        send_message(socket, bet_str)

    send_message(socket, END_MESSAGE)

def recv_batch_bet_message(socket: socket.socket, lottery: Lottery):
    header = recv_all(socket, HEADER_SIZE)
    if not header:
        raise ConnectionError("Connection error caused header to not be received")

    size = int.from_bytes(header)
    response = recv_all(socket, size)

    if not response:
       raise ValueError("Empty batch message")
    
    batchResponse = response.decode()
    if batchResponse == END_MESSAGE:
        return None

    batch = batchResponse.split("\n")
    return [bet_from_response(bet) for bet in batch]

def send_batch_succeeded(socket: socket.socket):
    send_message(socket, ACK_MESSAGE)

def send_batch_failed(socket: socket.socket):
    send_message(socket, NACK_MESSAGE)

def bet_from_response(bet_response: str):
    bet = bet_response.split(",")
    if len(bet) != BET_ARGUMENTS:
        raise ValueError("Incorrect number of fields for Bet")
    return Bet(int(bet[0]), bet[1], bet[2], int(bet[3]), bet[4], int(bet[5]))

def create_winner_message(winner: Bet):
    bet = [winner.first_name, winner.last_name, str(winner.document), winner.birthdate, str(winner.number)]
    return ",".join(bet)