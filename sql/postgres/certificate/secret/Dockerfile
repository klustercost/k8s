FROM python:3.13-slim-trixie  AS build

WORKDIR /app

COPY *.py ./
COPY requirements.txt .

RUN pip3 install --upgrade pip
RUN pip install -r ./requirements.txt
RUN rm -rf requirements.txt

## Build
FROM gcr.io/distroless/python3-debian13:latest

COPY --from=build /app /app
COPY --from=build /usr/local/lib/python3.13/site-packages /usr/local/lib/python3.13/site-packages

WORKDIR /app
ENV PYTHONPATH=/usr/local/lib/python3.13/site-packages

USER nonroot:nonroot

CMD ["main.py"]
