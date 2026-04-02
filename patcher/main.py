import logging
from requests import get
from os import environ
import jsonpath
import json
from kubernetes import client, config

logging.basicConfig(level=logging.INFO)
config.load_kube_config()
v1 = client.CoreV1Api()

def get_data():
    try:
        response = get( environ.get("URI_LOCATION") )
        if response.status_code == 200:
            return jsonpath.findall(environ.get("JSON_PATH_EXTRACT"),response.text)
    except:
        pass    
    return [None]

def patch(name,namespace,values):
    annotations = [
        {
            'op': 'replace',
            'path': '/spec/loadBalancerSourceRanges',
            'value': values
        }
    ]
    return v1.patch_namespaced_service(name=name, namespace=namespace, body=annotations)    

if __name__ == "__main__":
    imperative_values = json.loads(environ.get("EXTRA_ITEMS_JSON"))
    ipv4 = [ x for x in get_data() if ':' not in x] 

    patch_result = patch(
        name=environ.get("SERVICE_TO_PATCH"), 
        namespace=environ.get("NAMESPACE"),
        values = ipv4 + imperative_values)
    
    logging.info(f"the result of the patch is {patch_result}")
