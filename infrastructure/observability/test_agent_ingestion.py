"""Exercise the rendered filelog agent with synthetic CRI logs in Docker."""
import argparse
import datetime
import json
from pathlib import Path
import subprocess
import tempfile
import time
import urllib.request
import uuid

import yaml


def run(config_path, image):
    name = "bank-log-smoke-" + uuid.uuid4().hex[:12]
    config = yaml.safe_load(Path(config_path).read_text(encoding="utf-8"))
    # Use the actual receiver/processors with a debug sink, avoiding real Loki data.
    config["exporters"] = {"debug": {"verbosity": "detailed"}}
    config["service"]["pipelines"]["logs"]["exporters"] = ["debug"]
    selector = config["receivers"]["filelog/bank"]["include"][0]
    namespace = selector.split("/")[4].removesuffix("_*")
    container = "payment-service"
    try:
        with tempfile.TemporaryDirectory(prefix="bank-log-agent-") as temporary:
            root = Path(temporary)
            filename = root / "pods" / (namespace + "_smokepod_11111111-1111-4111-8111-111111111111") / container / "0.log"
            filename.parent.mkdir(parents=True)
            filename.touch()
            config_file = root / "agent.yaml"
            config_file.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
            subprocess.run([
                "docker", "run", "-d", "--name", name, "--user", "0", "--read-only",
                "--tmpfs", "/var/lib/otel-offsets:mode=700",
                "-p", "127.0.0.1::13133",
                "-v", str(config_file) + ":/etc/bank.yaml:ro,Z",
                "-v", str(root / "pods") + ":/var/log/pods:ro,Z",
                image, "--config=/etc/bank.yaml",
            ], check=True, capture_output=True)
            binding = subprocess.check_output(["docker", "port", name, "13133/tcp"], text=True).strip()
            endpoint = "http://127.0.0.1:" + binding.rsplit(":", 1)[1]
            for attempt in range(100):
                try:
                    with urllib.request.urlopen(endpoint, timeout=1) as response:
                        if response.status == 200:
                            break
                except (OSError, ValueError):
                    pass
                time.sleep(.2)
            else:
                raise RuntimeError("filelog agent did not become ready")
            # start_at=end must discover the empty file before appending messages.
            time.sleep(1)
            with filename.open("a", encoding="utf-8") as stream:
                for level in ("INFO", "WARN", "ERROR", "FATAL"):
                    message = {
                        "serviceName": container, "version": "smoke-test", "environment": "test",
                        "level": level, "message": "synthetic agent check",
                        "correlationId": "11111111-1111-4111-8111-111111111111",
                        "traceId": "a" * 32, "spanId": "b" * 16,
                    }
                    now = datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z")
                    stream.write(now + " stdout F " + json.dumps(message) + "\n")
            required = [
                "SeverityNumber: Info(9)", "SeverityNumber: Warn(13)", "SeverityNumber: Error(17)",
                "SeverityText: WARNING", "Trace ID: " + "a" * 32, "Span ID: " + "b" * 16,
                "correlationId: Str(11111111-1111-4111-8111-111111111111)",
            ]
            for attempt in range(100):
                output = subprocess.check_output(["docker", "logs", name], stderr=subprocess.STDOUT, text=True)
                if all(value in output for value in required):
                    print("Filelog ingestion: INFO/WARNING/ERROR, correlation, native trace/span IDs verified")
                    return
                time.sleep(.2)
            raise RuntimeError("expected severity/trace metadata missing from collected logs")
    except Exception:
        diagnostic = subprocess.run(["docker", "logs", name], text=True, capture_output=True)
        print(diagnostic.stdout + diagnostic.stderr)
        raise
    finally:
        subprocess.run(["docker", "rm", "-f", name], capture_output=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--image", required=True)
    arguments = parser.parse_args()
    run(arguments.config, arguments.image)
