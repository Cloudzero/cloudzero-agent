select
    distinct cluster_name,
    GET(labels, '__name__')::string as metric_name
from
    public.prometheus_staging
where
    organization_id = '39bead45-5e64-48b0-94b5-0d349b5ca6ef'
    and usage_date >= '2024-10-01 01'
order by
    cluster_name