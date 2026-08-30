import socket
import logger
import safe_socket
from communication.messages import recv_bet_message, bet_from_response
from src_frozen.lottery import Lottery, Bet
import lottery_bet

END_MESSAGE = "END"

class Server:
    def __init__(self, server_host: str, server_port: int, server_lottery: Lottery) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.server_lottery = server_lottery

    def _handle_client(self, client_socket):
        action = "handle-client"
        bets = []
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                bet_response = recv_bet_message(
                    client_socket,
                )

                if bet_response == END_MESSAGE:
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "bets-finished"
                    )
                    break

                bet = bet_from_response(bet_response)
                if not bet:
                    logger.error(
                        action,
                        logger.LogResult.fail,
                        "bet-not-received",
                    )
                    return
                bets.append(bet)

            self.server_lottery.store_bets(bets)

            winners = lottery_bet.get_winner(self.server_lottery)

            lottery_bet.send_winners(client_socket, winners)
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail,
            )
            raise e

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                with client_socket:
                    self._handle_client(client_socket)
