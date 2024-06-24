# How to Validate Deployment

This guide provides instructions on how to validate the deployment of the Helm chart. As part of the deployment, a container is started to execute environmental validations. This guide outlines how to use these to gather insight into if everything is ready post deployment.

## Steps to Validate the Deployment

1. **Get the Pod Name**: After deploying the Helm chart, you need to identify the pod running the `env-validator`. Use the following command to list all pods in the namespace where you have deployed the chart:

    ```sh
    kubectl -n cloudzero-agent get pods
    ```

The output of this should be similar to as follows, with your specific helm release name (r1):
```bash
NAME                                          READY   STATUS        RESTARTS   AGE
r1-cloudzero-agent-server-6dc588f9cb-zfvr7   0/2     Terminating   0          31s
r1-prometheus-node-exporter-69kg8            1/1     Running       0          5s
r1-kube-state-metrics-79894f6c55-q5rq4       0/1     Running       0          5s
r1-cloudzero-agent-server-6dc588f9cb-qqqm9   0/2     Init:0/1      0          5s
``` 


2. **Check Validation Results**: Once you have the pod name, you can check the validation results by viewing the logs of the `env-validator` container. Replace `<pod-name>` with the actual pod name obtained from the previous step:

    ```sh
    kubectl -n cloudzero-agent logs -f -c env-validator <pod-name>
    ```

### Example Output

The expected output of the `env-validator` will look something like the following. 

```plaintext
$ kubectl -n cloudzero-agent logs -f -c env-validator r1-cloudzero-agent-server-6dc588f9cb-zfvr7
Validation starting...
Collecting requests==2.32.3 (from -r requirements.txt (line 1))
Downloading requests-2.32.3-py3-none-any.whl.metadata (4.6 kB)
Collecting charset-normalizer<4,>=2 (from requests==2.32.3->-r requirements.txt (line 1))
Downloading charset_normalizer-3.3.2-cp312-cp312-manylinux_2_17_aarch64.manylinux2014_aarch64.whl.metadata (33 kB)
Collecting idna<4,>=2.5 (from requests==2.32.3->-r requirements.txt (line 1))
Downloading idna-3.7-py3-none-any.whl.metadata (9.9 kB)
Collecting urllib3<3,>=1.21.1 (from requests==2.32.3->-r requirements.txt (line 1))
Downloading urllib3-2.2.1-py3-none-any.whl.metadata (6.4 kB)
Collecting certifi>=2017.4.17 (from requests==2.32.3->-r requirements.txt (line 1))
Downloading certifi-2024.6.2-py3-none-any.whl.metadata (2.2 kB)
Downloading requests-2.32.3-py3-none-any.whl (64 kB)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 64.9/64.9 kB 4.9 MB/s eta 0:00:00
Downloading certifi-2024.6.2-py3-none-any.whl (164 kB)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 164.4/164.4 kB 4.7 MB/s eta 0:00:00
Downloading charset_normalizer-3.3.2-cp312-cp312-manylinux_2_17_aarch64.manylinux2014_aarch64.whl (137 kB)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 137.3/137.3 kB 13.3 MB/s eta 0:00:00
Downloading idna-3.7-py3-none-any.whl (66 kB)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 66.8/66.8 kB 9.1 MB/s eta 0:00:00
Downloading urllib3-2.2.1-py3-none-any.whl (121 kB)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 121.1/121.1 kB 12.0 MB/s eta 0:00:00
Installing collected packages: urllib3, idna, charset-normalizer, certifi, requests
Successfully installed certifi-2024.6.2 charset-normalizer-3.3.2 idna-3.7 requests-2.32.3 urllib3-2.2.1
Running validations...
http://r1-kube-state-metrics:8080/ not ready yet
------------------------------------------------------------
CHECK                                              RESULT
external_connectivity_available                    success
kube_state_metrics_available                       success
prometheus_node_exporter_available                 success
------------------------------------------------------------
Validator finished.
```

If all checks show `success`, the deployment is validated successfully.
