## Build
FROM python:3.12-bookworm AS build

WORKDIR /app

COPY *.py ./
COPY requirements.txt ./

RUN pip3 install --upgrade pip
RUN pip install -r ./requirements.txt

CMD ["main.py"]
