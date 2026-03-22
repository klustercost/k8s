import os
import logging

LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
log = logging.getLogger("wa-webhook")
logging.basicConfig(
    level=LOG_LEVEL,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log.setLevel(logging.INFO)