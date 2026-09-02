import socket
import logger
import threading
from communication.messages import recv_batch_bet_message, send_messages, send_batch_failed, send_batch_succeeded
from .utils import get_winner
from lottery import Lottery

class Server:
    def __init__(self, server_host: str, server_port: int, server_lottery: Lottery, agency_quorum_min: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.server_lottery = server_lottery
        self.agency_quorum_min = agency_quorum_min
        self.agencies_finished = 0
        self.lock = threading.Lock()
        self.cond_var = threading.Condition()
        self.threads = []

    def _handle_client(self, client_socket):
        action = "handle-client"
        agency_id = None
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                try:
                    bets = recv_batch_bet_message(client_socket)
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
                if agency_id == None:
                    agency_id = bets[0].agency_id
                with self.lock:
                    self.server_lottery.store_bets(bets)
                send_batch_succeeded(client_socket)

            with self.cond_var:
                self.agencies_finished += 1
                if self.quorum_reached():
                    self.cond_var.notify_all()
                self.cond_var.wait_for(self.quorum_reached)
            
            winners = get_winner(self, agency_id)
            send_messages(client_socket, winners)
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail,
            )
            raise e
        finally:
            client_socket.close()

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                self.threads = [t for t in self.threads if t.is_alive()]    
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                client_thread = threading.Thread(target=self._handle_client, args=(client_socket,))
                client_thread.start()
                self.threads.append(client_thread)
    def quorum_reached(self):
        return self.agency_quorum_min <= self.agencies_finished