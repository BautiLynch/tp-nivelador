import socket
import logger
from communication.messages import recv_batch_bet_message, send_messages, send_batch_failed, send_batch_succeeded
from .utils import get_winner
from lottery import Lottery

class Server:
    def __init__(self, server_host: str, server_port: int, server_lottery: Lottery) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.server_lottery = server_lottery

    def _handle_client(self, client_socket):
        action = "handle-client"
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                try:
                    bets = recv_batch_bet_message(client_socket, self.server_lottery)
                except ConnectionError as error:
                    logger.error(
                        action,
                        logger.LogResult.fail,
                        "connection-error",
                        "err",
                        error,
                    )
                    return
                except:
                    send_batch_failed(client_socket)
                    continue
                if not bets:
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "bets-finished"
                    )
                    break
                self.server_lottery.store_bets(bets)
                send_batch_succeeded(client_socket)
                

            winners = get_winner(self.server_lottery)
            send_messages(client_socket, winners)
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
