def get_winner(server, agency_id):
    lottery = server.server_lottery
    winners = []
    with server.lock:
        for bet in lottery.load_bets():
            if lottery.has_won(bet) and bet.agency_id == agency_id: 
                winners.append(bet)


    return winners