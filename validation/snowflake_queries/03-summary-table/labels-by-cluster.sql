USE DATABASE live_billing;
SET
    cluster_name = 'azure-cirrus-prom-only'; -- update cluster as needed
SET
    lookback_time = '2024-10-01 10'::timestamp_tz;
ALTER SESSION
SET
    TIMEZONE = 'UTC';
SELECT
    cloud_account_id,
    cluster_name,
    metrics_hour,
    tags
FROM
    cz_prometheus_external."02FA7D30_C3DE_4E0A_8F1E_2DE120E7FD23" -- CZ LIVE ORG
WHERE
    1 = 1
    and metrics_hour >= $lookback_time
    and cluster_name = $cluster_name
ORDER BY
    cloud_account_id,
    cluster_name,
    metrics_hour