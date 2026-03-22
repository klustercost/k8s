import uvicorn
from dotenv import load_dotenv
import os

load_dotenv(override=False)

if __name__ == "__main__":
    test = os.getenv("HOST_PORT")
    uvicorn.run("src.hook:app", host="0.0.0.0", port=int(os.getenv("HOST_PORT")), reload=True)