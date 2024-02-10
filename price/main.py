from flask import Flask
from flask import request
import json
import logging
import azure
from enum import Enum
import os

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

app = Flask(__name__)

@app.route('/get')
def get():
    try:
        region = request.args.get('region')
        sku = request.args.get('sku')

        if not region:
            raise Exception("Missing region")
        if not sku:
            raise Exception("Missing sky")

        return json.dumps(get_api().query(region, sku))
    except Exception as Ex:
        return str(Ex)

@app.route('/about')
def about():
    return "{\"about\":\"klustercost Price Server\"}"

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    from waitress import serve
    logging.info('This is the klustercost price server')
    serve(app, host="0.0.0.0", port=5001)