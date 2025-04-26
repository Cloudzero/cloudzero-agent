# Testing the Validator in the CloudZero Agent Server Container

This document provides instructions on how to test the CloudZero Agent Validator within the `cloudzero-agent-server` container.

## Prerequisites
- Ensure you have access to a Kubernetes cluster where the CloudZero Agent is installed.
- Ensure the `cloudzero-agent` is installed on the cluster.

## Steps to Test the Validator

1. **Access the CloudZero Agent Server Container**  
    Use the following command to access the `cloudzero-agent-server` container within the CloudZero Agent pod:
    ```bash
    kubectl -n cloudzero-agent exec -it <pod-name> -c cloudzero-agent-server -- sh
    ```
    Replace `<pod-name>` with the name of the running `cloudzero-agent-server` pod.

2. **Navigate to the Checks Directory**  
    Once inside the container, navigate to the `/checks/` directory:
    ```bash
    cd /checks/
    ```

3. **Run the Validator**  
    Use the `cloudzero-agent-validator` binary to run specific checks. For example:

    ```bash
    ./cloudzero-agent-validator d run -f /checks/app/config/validator.yml --check webhook_server_reachable
    ```

4. **Verify Results**  
    After running the validator, review the output for any errors or issues. The results will indicate whether the specified checks passed or failed.

## Notes
- Your data will be available in the CloudZero portal after 48 hours.
- For additional help, use the `--help` flag:
    ```bash
    ./cloudzero-agent-validator --help
    ```
