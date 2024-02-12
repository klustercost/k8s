## Build
FROM python:3.12-bookworm AS build

WORKDIR /app

COPY *.py ./
COPY requirements.txt ./

RUN pip3 install --upgrade pip
RUN pip install -r ./requirements.txt

# Make main.py executable
RUN chmod +x main.py

CMD ["python", "./main.py"]
