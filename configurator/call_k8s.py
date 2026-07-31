import json
import logging
from typing import Generator
from kubernetes import client as k8s_client, config as k8s_config
from config import config

_KLUSTERCOST_OBJECT_CLASS = 'klustercost.cloud/object-class'

if config.run_local is None or config.run_local.lower() != 'true':
    k8s_config.load_incluster_config()
else:
    k8s_config.load_kube_config()

v1 = k8s_client.CoreV1Api()

def transformers() -> Generator[tuple[str, str]]:
    for configmap in json.loads(config.monitor):
        try:
            cm = v1.read_namespaced_config_map(namespace=config.namespace,name=configmap)
            yield (cm.data.get('labels.jsonata'), cm.metadata.annotations.get(_KLUSTERCOST_OBJECT_CLASS))
        except k8s_client.exceptions.NotFoundException as e:
            logging.error(f"Error occurred while fetching ConfigMap {configmap}: {e}")
            continue
