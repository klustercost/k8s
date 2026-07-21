import os
import logging
from dotenv import load_dotenv

load_dotenv(override=False)

class _GetLogLevel:
    def __get__(self, obj, objtype=None):
        log_level = os.environ.get("log_level", None)
        if log_level == None:
            return logging.INFO
        return logging.getLevelName(log_level.upper())

class _GetNamespace:
    def __get__(self, obj, objtype=None):
        if obj.run_local is None or obj.run_local.upper() != "TRUE":
            with open('/var/run/secrets/kubernetes.io/serviceaccount/namespace', 'r') as namespace_file:
                return namespace_file.read().rstrip()    
        else:
            return os.environ.get('namespace')

class _Config:
    def __init__(self, *environment_variables):        
        for var in environment_variables:
            setattr(self, var, os.environ.get(var, ""))

    log_level = _GetLogLevel()
    namespace = _GetNamespace()

config = _Config('db_host', 'db_database', 'db_user', 'db_password', 'db_port', 'server_port','ddl_endpoint', 'run_local','monitor')
