## Build
FROM python:3.11-bookworm  AS build

WORKDIR /app

COPY *.py ./
COPY requirements.txt ./

RUN pip3 install --upgrade pip
RUN pip install -r ./requirements.txt

## Build
FROM gcr.io/distroless/python3-debian12:latest

COPY --from=build /lib/x86_64-linux-gnu/libpq.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libldap-2.5.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/liblber-2.5.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libsasl2.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libgnutls.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libp11-kit.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libidn2.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libunistring.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libtasn1.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libnettle.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libhogweed.s* /lib/x86_64-linux-gnu/
COPY --from=build /lib/x86_64-linux-gnu/libgmp.s* /lib/x86_64-linux-gnu/

COPY --from=build /app /app
COPY --from=build /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages

WORKDIR /app
ENV PYTHONPATH=/usr/local/lib/python3.11/site-packages

USER nonroot:nonroot

CMD ["main.py"]