import os
from dotenv import load_dotenv

class Config:
    def __init__(self, *environment_variables):
        load_dotenv()
        for var in environment_variables:
            setattr(self, var, os.environ.get(var, ""))
