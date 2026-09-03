import socket
import logger
import threading
import signal
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
        self.sigterm_event = threading.Event()
        signal.signal(signal.SIGTERM, self.sigterm_handler)

    def sigterm_handler(self, signum, frame):
        self.sigterm_event.set()

    def _handle_client(self, client_socket):
        action = "handle-client"
        agency_id = None
        try:
            logger.info(action, logger.LogResult.in_progress)
            while not self.sigterm_event.is_set():
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
                    if self.sigterm_event.is_set():
                        return
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

            if self.sigterm_event.is_set():
                return

            with self.cond_var:
                self.agencies_finished += 1
                if self.quorum_reached():
                    self.cond_var.notify_all()
                self.cond_var.wait_for(self.necessary_condition)
                if self.sigterm_event.is_set():
                    return
            
            winners = get_winner(self, agency_id)
            if self.sigterm_event.is_set():
                return
            send_messages(client_socket, winners)
        except Exception as e:
            if not self.sigterm_event.is_set():
                logger.error(
                    action, logger.LogResult.fail, "err", e
                )
        finally:
            client_socket.close()

    def run(self):
        action = "accept-connection"
        client_sockets = []
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
                server_socket.bind((self.server_host, self.server_port))
                server_socket.listen()
                server_socket.settimeout(0.5)
                while not self.sigterm_event.is_set():
                    try:
                        logger.info(action, logger.LogResult.in_progress)
                        client_socket, _ = server_socket.accept()
                    except socket.timeout:
                        continue
                    except Exception as e:
                        if self.sigterm_event.is_set():
                            break
                        logger.error(action, logger.LogResult.fail)
                        raise e
                        
                    if self.sigterm_event.is_set():
                        client_socket.close()
                        break

                    client_sockets.append(client_socket)
                    logger.info(action, logger.LogResult.success)

                    client_thread = threading.Thread(target=self._handle_client, args=(client_socket,))
                    client_thread.start()
                    self.threads.append(client_thread)
        finally:
            self.sigterm_event.set()

            with self.cond_var:
                self.cond_var.notify_all()

            for client_sock in client_sockets:
                try:
                    client_sock.shutdown(socket.SHUT_RDWR)
                except:
                    pass

                try:
                    client_sock.close()
                except:
                    pass

            for thread in self.threads:
                thread.join()

            self.threads.clear()

    def quorum_reached(self):
        return self.agency_quorum_min <= self.agencies_finished
    
    def necessary_condition(self):
        return self.quorum_reached() or self.sigterm_event.is_set()