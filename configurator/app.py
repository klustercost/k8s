from fastapi import FastAPI
from requests import post, exceptions
import json
from types import SimpleNamespace
from config import config
from call_db import call_db
import call_k8s

app = FastAPI()

@app.get('/about')
def about():
    return {"about": "klustercost configurator"}

@app.get('/health')
def health():
    return {"status": "healthy"}

@app.post('/update_persistence')
def persistence_update():
    return_val = []
    for enum_transformer in call_k8s.transformers():
        try:
            response = post(
                config.ddl_endpoint,
                json={
                    "jsonata": enum_transformer[0],
                    "table_name": f'tbl_{enum_transformer[1]}s',
                    "schema": config.db_database,
                    "index_columns": ["uid", "namespace", "node", "app.name", "app.component"]
                }
            )
        except exceptions.RequestException as e:
            return_val.append({"status": "Failed", "error": "Failed to update persistence", "details": str(e), "transformer": enum_transformer[1]})
            continue

        if response.status_code != 200:
            try:
                return_val.append({"status": "Failed", "error": "Failed to update persistence", "details": json.loads(response.text), "transformer": enum_transformer[1]})
                continue
            except json.decoder.JSONDecodeError as e:
                return_val.append({"status": "Failed", "error": "Failed to update persistence", "details": response.text, "transformer": enum_transformer[1]})
                continue
        
        response_object = json.loads(response.text, object_hook=SimpleNamespace)
        if "success" != response_object.status:
            return_val.append({"status": "Failed", "error": "Failed to update persistence", "details": response_object.status, "transformer": enum_transformer[1]})
            continue

        try:
            call_db().call(response_object.ddl)
        except Exception as e:
            return_val.append({"status": "Failed", "error": "Failed to update persistence", "details": str(e), "transformer": enum_transformer[1]})
            continue
        
        return_val.append({"status": "OK", "details": "Persistence updated successfully", "transformer": enum_transformer[1]})

    return return_val
