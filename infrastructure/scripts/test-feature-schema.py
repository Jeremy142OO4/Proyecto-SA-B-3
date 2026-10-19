#!/usr/bin/env python3
"""Exercise the deployment SQL twice on disposable PostgreSQL; never uses real DBs."""
from pathlib import Path
import subprocess
import sys
import time
import uuid

ROOT = Path(__file__).resolve().parents[2]
CONTAINER = "bank-schema-test-" + uuid.uuid4().hex[:12]


def docker(*args, **kwargs):
    result = subprocess.run(["docker", *args], text=True, capture_output=True, **kwargs)
    if result.returncode:
        detail = result.stderr.strip() or result.stdout.strip()
        raise RuntimeError(f"docker {' '.join(args)} failed ({result.returncode}): {detail}")
    return result


def sql(database, statement, fail=False):
    result = subprocess.run(
        ["docker", "exec", "-i", CONTAINER, "psql", "-h", "127.0.0.1", "-U", "postgres", "-d", database,
         "--no-psqlrc", "--set=ON_ERROR_STOP=1", "--single-transaction", "-At"],
        input=statement, text=True, capture_output=True)
    if fail:
        assert result.returncode != 0, "injected SQL failure was not detected"
    elif result.returncode:
        raise RuntimeError(result.stderr)
    return result.stdout.strip()


def main():
    try:
        docker("run", "-d", "--rm", "--name", CONTAINER, "--network=none",
               "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "postgres:17-alpine")
        # The image first starts a temporary server on a Unix socket, then stops
        # it before starting the final server. TCP only becomes available on the
        # final server; socket readiness can therefore race with that shutdown.
        for attempt in range(60):
            ready = subprocess.run(["docker", "exec", CONTAINER, "pg_isready", "-h", "127.0.0.1", "-U", "postgres", "-d", "postgres"], capture_output=True)
            if ready.returncode == 0:
                break
            time.sleep(1)
        else:
            raise RuntimeError("test PostgreSQL did not become ready")
        script = (ROOT / "infrastructure/scripts/prepare-telemetry-schema.sh").read_text(encoding="utf-8")
        entries = script.split("done <<'SERVICES'\n", 1)[1].split("\nSERVICES", 1)[0]
        for index, line in enumerate(entries.splitlines()):
            service, _, *names = line.split()
            database = "schema_test_" + str(index)
            docker("exec", CONTAINER, "createdb", "-h", "127.0.0.1", "-U", "postgres", database)
            folder = ROOT / "services" / service / "database/migrations"
            first = int(names[0].split("_", 1)[0])
            for migration in sorted(folder.glob("*.up.sql")):
                if int(migration.name.split("_", 1)[0]) < first:
                    sql(database, migration.read_text(encoding="utf-8"))
            statements = "SELECT pg_advisory_xact_lock(hashtext('bank-usac-schema'));\n"
            statements += "\n".join((folder / (name + ".up.sql")).read_text(encoding="utf-8") for name in names)
            if service == "account-service":
                sql(database, "INSERT INTO cuentas(id_cliente,numero_cuenta,tipo_cuenta,saldo_centavos) VALUES(gen_random_uuid(),'123456789','AHORRO',12345);")
                sql(database, statements + "\nSELECT 1/0;", fail=True)
                assert sql(database, "SELECT to_regclass('anomalias_cuenta') IS NULL;") == 't', 'failed migration did not roll back'
            for _ in range(2):
                sql(database, statements)
            if service == "account-service":
                assert sql(database, "SELECT saldo_centavos FROM cuentas WHERE numero_cuenta='123456789';") == '12345'
                assert sql(database, "SELECT count(*) FROM anomalias_cuenta;") == '0'
            elif service == "service-notification-audit":
                sql(database, "INSERT INTO notification_logs(correlation_id,recipient,notification_type,subject,body_summary) VALUES(gen_random_uuid(),'test@example.test','TEST','test','test');")
                assert sql(database, "SELECT severity || ':' || details::text FROM notification_logs;") == 'INFO:{}'
            elif service == "service-customer":
                assert sql(database, "SELECT count(*) FROM customer_lifecycle_events;") == '0'
            print(service + ': initial migration and rerun OK')
    except Exception:
        logs = subprocess.run(["docker", "logs", CONTAINER], text=True, capture_output=True)
        print("Test PostgreSQL container logs:", file=sys.stderr)
        print(logs.stdout + logs.stderr, file=sys.stderr)
        raise
    finally:
        subprocess.run(["docker", "rm", "-f", CONTAINER], capture_output=True)


if __name__ == "__main__":
    main()
