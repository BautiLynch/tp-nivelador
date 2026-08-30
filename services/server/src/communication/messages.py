import socket
from safe_socket import recv_all, send_all
from src_frozen.lottery.bet import Bet 

HEADER_SIZE = 4
BET_ARGUMENTS = 6

def recv_bet_message(socket: socket.socket):
    header = recv_all(socket, HEADER_SIZE)
    if not header:
        return None

    size = int.from_bytes(header, "big", False)
    response = recv_all(socket, size)

    if not response:
        return None
    
    return response.decode() 


def send_winner_message(socket: socket.socket, winner: str):
    bytes_winner = winner.encode()
    size = len(bytes_winner)
    header_bytes = size.to_bytes(HEADER_SIZE, "big", False)
    send_all(socket, header_bytes)
    send_all(socket, bytes_winner)


def bet_from_response(bet_response: str):
    if not bet_response:
        return None
    bet = bet_response.split(",")
    if len(bet) != BET_ARGUMENTS:
        return None
    return Bet(int(bet[0]), bet[1], bet[2], int(bet[3]), bet[4], int(bet[5]))

def create_winner_message(winner: str):
    bet = [winner.first_name, winner.last_name, str(winner.document), winner.birthdate, str(winner.number)]
    return ",".join(bet)