from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta
import psycopg2
import clickhouse_connect

def extract_crm(**kwargs):
    conn = psycopg2.connect(
        host="crm-db", 
        dbname="crm",
        user="crm_user",
        password="crm_password"
    )
    cur = conn.cursor()
    cur.execute("SELECT user_id, name, region FROM customers;")
    rows = cur.fetchall()
    conn.close()
    return rows


def extract_telemetry(**kwargs):
    client = clickhouse_connect.get_client(
        host="clickhouse",
        username="default",
        password="root"
    )
    query = """
        SELECT user_id, 
               AVG(metric) as avg_metric, 
               MAX(metric) as max_metric
        FROM telemetry
        GROUP BY user_id
    """
    result = client.query(query)
    return result.result_rows

def transform_and_load(**kwargs):
    ti = kwargs["ti"]
    crm_data = ti.xcom_pull(task_ids="extract_crm")
    telemetry_data = ti.xcom_pull(task_ids="extract_telemetry")

    telemetry_map = {row[0]: row[1:] for row in telemetry_data}

    client = clickhouse_connect.get_client(
        host="clickhouse",
        username="default",
        password="root"
    )

    client.command("""
        CREATE TABLE IF NOT EXISTS user_reports (
            user_id UInt64,
            name String,
            region String,
            avg_metric Float64,
            max_metric Float64
        ) ENGINE = MergeTree()
        ORDER BY user_id
    """)

    batch = []
    for user_id, name, region in crm_data:
        avg_metric, max_metric = telemetry_map.get(user_id, (0, 0))
        batch.append((user_id, name, region, avg_metric, max_metric))

    client.insert(
        "user_reports",
        batch,
        column_names=["user_id", "name", "region", "avg_metric", "max_metric"]
    )

default_args = {
    "owner": "airflow",
    "depends_on_past": False,
    "retries": 1,
    "retry_delay": timedelta(minutes=5),
}

with DAG(
    "etl_crm_telemetry",
    default_args=default_args,
    description="ETL из CRM и телеметрии в витрину user_reports (ClickHouse)",
    schedule="@daily", 
    start_date=datetime(2023, 1, 1),
    catchup=False,
    tags=["etl", "clickhouse"],
) as dag:

    extract_crm_task = PythonOperator(
        task_id="extract_crm",
        python_callable=extract_crm,
    )

    extract_telemetry_task = PythonOperator(
        task_id="extract_telemetry",
        python_callable=extract_telemetry,
    )

    transform_and_load_task = PythonOperator(
        task_id="transform_and_load",
        python_callable=transform_and_load,
    )

    [extract_crm_task, extract_telemetry_task] >> transform_and_load_task
