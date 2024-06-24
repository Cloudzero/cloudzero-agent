#!/usr/bin/env python3

import os
import requests
import sys
import time
from typing import Dict, Tuple, Callable, List

MAX_RETRY = 12
RETRY_INTERVAL = 10


def check_service_availability(url: str, key: str) -> Tuple[str, str]:
    # only wait two minutes max
    for _ in range(MAX_RETRY):
        try:
            response = requests.get(url)
            response.raise_for_status()
            return (key, "success")
        except Exception:
            print(f"{url} not ready yet", file=sys.stderr)
            time.sleep(RETRY_INTERVAL)

    print(f"{key} not ready after {MAX_RETRY} retries", file=sys.stderr)
    return (key, "failure")


def check_external_connectivity() -> Tuple[str, str]:
    host: str = os.getenv("CZ_HOST", "api.cloudzero.com")
    url: str = f"http://{host}"
    return check_service_availability(url, "external_connectivity_available")


def check_kube_state_metrics() -> Tuple[str, str]:
    url: str = os.getenv("KMS_EP_URL", "http://kube-state-metrics:8080/")
    return check_service_availability(url, "kube_state_metrics_available")


def check_prometheus_node_exporter() -> Tuple[str, str]:
    url: str = os.getenv(
        "NODE_EXPORTER_EP_URL", "http://prometheus-node-exporter:9100/"
    )
    return check_service_availability(url, "prometheus_node_exporter_available")


validations: List[Callable[[], Tuple[str, str]]] = [
    check_external_connectivity,
    check_kube_state_metrics,
    check_prometheus_node_exporter,
]


def run_validations() -> Dict[str, str]:
    return {key: value for check in validations for key, value in [check()]}


def must_pass(results: Dict[str, str]):
    if any(result != "success" for result in results.values()):
        exit(1)


def print_results(results: Dict[str, str]) -> None:
    print("-" * 60)
    print("\n".join([f"{check:<50} {result:<10}" for check, result in results.items()]))
    print("-" * 60)


def main():
    results = run_validations()
    print_results(results)
    # TODO - post results to status API endpoint
    must_pass(results)


if __name__ == "__main__":
    main()
