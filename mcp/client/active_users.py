from time import time
from threading import Lock
from os import getenv

MCP_SESSION_TIMEOUT =  int(getenv("MCP_SESSION_TIMEOUT", "3600"))  # 1 hour in seconds

class _active_users:
    def __init__(self):
        self._lock = Lock()
        self._users : dict[str,(str,float)] = {}

    def get_last_request(self, user:str) -> str|None:
        with self._lock:
            if user not in self._users:
                return None
            user_data = self._users[user]
            if time() - user_data[1] > MCP_SESSION_TIMEOUT:
                del self._users[user]
                return None
            return user_data[0]
    
    def set_last_requert(self, user:str, request:str) -> None:
        with self._lock:
            self._users[user] = (request, time())

_active_users_singleton = _active_users()

def get_last_request(user:str) -> str|None:
    return _active_users_singleton.get_last_request(user)

def set_last_request(user:str, request:str):
    _active_users_singleton.set_last_requert(user, request)