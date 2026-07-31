import logging
import uvicorn
from config import config
from app import app

if __name__ == "__main__":
    logging.basicConfig(level=config.log_level)
    logging.info('This is the klustercost configurator')
    uvicorn.run(app, host="0.0.0.0", port=config.server_port)
