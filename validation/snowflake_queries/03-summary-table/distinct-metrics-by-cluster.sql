set organization_id = 'your-org-id';
set lookback_time = '2024-10-01 10'::timestamp_tz;

select
    distinct cluster_name,
    GET(labels, '__name__')::string as metric_name
from
    public.prometheus_staging
where
    organization_id = $organization_id
    and usage_date >= $lookback_time
order by
    cluster_name
