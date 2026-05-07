import logging
from kubernetes import client, config
from os import environ
from time import sleep
from load_dotenv import load_dotenv

logging.basicConfig(level=logging.INFO)

load_dotenv()

SECRET_TO_WATCH=environ.get('SECRET_TO_WATCH')
TIMEOUT=environ.get('TIMEOUT', 60)

if environ.get('RUN_LOCAL') is None:
    config.load_incluster_config()
    with open('/var/run/secrets/kubernetes.io/serviceaccount/namespace', 'r') as namespace_file:
        NAMESPACE = namespace_file.read().rstrip()    
else:
    config.load_kube_config()
    NAMESPACE=environ.get('NAMESPACE')

if __name__ == "__main__":
    v1 = client.CoreV1Api()
    logging.info(f"Checking on {SECRET_TO_WATCH} in namespace {NAMESPACE}:")
    try:
        current_version = v1.read_namespaced_secret(name=SECRET_TO_WATCH, namespace=NAMESPACE).metadata.resource_version
    except client.exceptions.ApiException as e:
        logging.error(f"Exception while fetching secret: {e}")
        current_version = None
    logging.info(f"Secret {SECRET_TO_WATCH} has current version: {current_version}")
    while True:
        sleep(TIMEOUT)
        if v1.read_namespaced_secret(name=SECRET_TO_WATCH, namespace=NAMESPACE).metadata.resource_version != current_version:
            logging.info(f"Secret {SECRET_TO_WATCH} has changed! Restarting...")
            break
        else:
            logging.debug(f"Secret {SECRET_TO_WATCH} unchanged.")
