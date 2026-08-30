from src_frozen.lottery import Lottery, Bet
import socket
from communication.messages import send_winner_message, create_winner_message

END_MESSAGE = "END"


def get_winner(lottery: Lottery):
    bets = lottery.load_bets()
    winners = []

    for bet in bets:
        if lottery.has_won(bet):
            winners.append(bet)

    return winners

def send_winners(socket: socket.socket, winners: list[Bet]):
    for winner in winners:
        bet_str = create_winner_message(winner)
        send_winner_message(socket, bet_str)

    send_winner_message(socket, END_MESSAGE)
    

