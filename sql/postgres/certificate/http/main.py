import os
from fastapi import FastAPI
from uvicorn import run
from load_dotenv import load_dotenv

load_dotenv()

app = FastAPI()

@app.get("/")
async def home():
    return {"message": "OK"}

if __name__ == "__main__":
    run("main:app", host="0.0.0.0", port=int(os.getenv("HOST_PORT")))