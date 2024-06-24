import os
import sys
import pytest
from unittest.mock import patch, MagicMock, Mock

# disable lint E402
# E402: module level import not at top of file
# Reason: The selected code is importing modules after some statements,
#         which triggers the E402 linting error. Disabling the
#         linting for this line will suppress the error.
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
from src.validate import (  # noqa: E402
    check_service_availability,
    check_external_connectivity,
    check_kube_state_metrics,
    check_prometheus_node_exporter,
    run_validations,
)


@pytest.fixture
def mock_requests_get(monkeypatch):
    mock_get = MagicMock()
    mock_get.return_value.status_code = 200
    monkeypatch.setattr("src.validate.requests.get", mock_get)
    monkeypatch.setattr("src.validate.MAX_RETRY", 1)
    monkeypatch.setattr("src.validate.RETRY_INTERVAL", 0.1)
    return mock_get


def test_check_service_availability_success(mock_requests_get, capfd):
    url = "http://example.com"
    key = "test_service"
    result, _ = check_service_availability(url, key)
    assert result == key
    assert not capfd.readouterr()[1]


def test_check_service_availability_failure(mock_requests_get, capfd):
    mock_requests_get.side_effect = Exception("Service not available")
    url = "http://example.com"
    key = "test_service"
    _, result = check_service_availability(url, key)
    assert result == "failure"
    assert capfd.readouterr()[1]


@patch("src.validate.check_service_availability")
def test_check_external_connectivity(mock_check_service_availability):
    mock_check_service_availability.return_value = (
        "external_connectivity_available",
        "success",
    )
    result = check_external_connectivity()
    assert result == ("external_connectivity_available", "success")


@patch("src.validate.check_service_availability")
def test_check_kube_state_metrics(mock_check_service_availability):
    mock_check_service_availability.return_value = (
        "kube_state_metrics_available",
        "success",
    )
    result = check_kube_state_metrics()
    assert result == ("kube_state_metrics_available", "success")


@patch("src.validate.check_service_availability")
def test_check_prometheus_node_exporter(mock_check_service_availability):
    mock_check_service_availability.return_value = (
        "prometheus_node_exporter_available",
        "success",
    )
    result = check_prometheus_node_exporter()
    assert result == ("prometheus_node_exporter_available", "success")


@patch(
    "src.validate.validations",
    [Mock(return_value=("test1", "success")), Mock(return_value=("test2", "failure"))],
)
def test_run_validations():
    result = run_validations()
    assert result == {"test1": "success", "test2": "failure"}
