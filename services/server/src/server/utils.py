from lottery import Lottery

def get_winner(lottery: Lottery):
    bets = lottery.load_bets()
    winners = []

    for bet in bets:
        if lottery.has_won(bet):
            winners.append(bet)

    return winners