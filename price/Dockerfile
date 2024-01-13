## Build
FROM python:3.9-buster  AS build

WORKDIR /app

COPY *.py ./
COPY requirements.txt ./

RUN pip3 install --upgrade pip
RUN pip install -r ./requirements.txt

## Build
FROM gcr.io/distroless/python3

COPY --from=build /app /app
COPY --from=build /usr/local/lib/python3.9/site-packages /usr/local/lib/python3.9/site-packages

WORKDIR /app
ENV PYTHONPATH=/usr/local/lib/python3.9/site-packages

USER nonroot:nonroot

ENTRYPOINT ["./main.py"]
