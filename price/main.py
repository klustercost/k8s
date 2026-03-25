from fastapi import FastAPI, Query
from fastapi.responses import JSONResponse
import uvicorn
import logging
import azure
from enum import Enum
import os
from dotenv import load_dotenv

load_dotenv(override=False)

class provider_type(Enum):
    AZURE = 1
    AWS = 2
    GCP = 3

def read_provider():
    provider_string = os.getenv("PROVIDER")
    if None == provider_string:
        raise EnvironmentError("No provider type set.")
    if provider_string.lower() == 'azure':
        return provider_type.AZURE
    if provider_string.lower() == 'aws':
        return provider_type.AWS
    if provider_string.lower() == 'gcp':
        return provider_type.GCP
    raise TypeError("Unknown provider type: " + provider_string)

def get_api():
    match read_provider():
        case provider_type.AZURE:
            return azure.price_api()
        case provider_type.AWS:
            raise NotImplementedError()
        case provider_type.GCP:
            raise NotImplementedError()

app = FastAPI()

@app.get('/get')
def get(region: str = Query(None), sku: str = Query(None), os: str = Query(None)):
    try:
        if not region:
            raise Exception("Missing region")
        if not sku:
            raise Exception("Missing sku")

        return get_api().query(region, sku, os)
    except Exception as Ex:
        return JSONResponse(content=str(Ex), status_code=400)

@app.get('/about')
def about():
    return {"about": "klustercost Price Server"}

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    logging.info('This is the klustercost price server')
    uvicorn.run(app, host="0.0.0.0", port=5001)