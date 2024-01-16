## Build
FROM python:3.12-bullseye  AS build

WORKDIR /app

COPY *.py ./
COPY requirements.txt ./

RUN pip3 install --upgrade pip
RUN pip install -r ./requirements.txt

## Build
FROM gcr.io/distroless/python3-debian12:latest

COPY --from=build /app /app
COPY --from=build /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages

WORKDIR /app
ENV PYTHONPATH=/usr/local/lib/python3.12/site-packages

USER nonroot:nonroot

CMD ["main.py"]
