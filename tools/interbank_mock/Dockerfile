FROM python:3.13-alpine
WORKDIR /app
COPY tools/interbank_mock/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt && adduser -D -u 10001 mock
COPY tools /app/tools
COPY contracts/interbank /app/contracts/interbank
RUN mkdir -p /app/state && chown mock /app/state
USER 10001:10001
ENTRYPOINT ["python", "-m", "tools.interbank_mock.app"]
CMD ["--database", "/app/state/mock.sqlite3"]
