FROM python:3.14-slim-bookworm

RUN groupadd --gid 1000 appuser && \
    useradd --uid 1000 --gid 1000 --create-home appuser

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY my_server.py system_prompt.txt ./

USER appuser

EXPOSE 8000

CMD ["python", "my_server.py"]
