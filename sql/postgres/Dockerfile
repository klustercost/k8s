FROM debian:trixie-slim AS deb_extractor

RUN cd /tmp && \
    apt-get update && apt-get download \
        postgresql-client-17 libc6 liblz4-1 libpq5 libreadline8t64 libzstd1 postgresql-client-common sensible-utils perl libperl5.40 perl-base perl-modules-5.40 libcrypt1 libgssapi-krb5-2 libldap2 libtinfo6 libkrb5-3 libk5crypto3 libcom-err2 libkrb5support0 libsasl2-2 libkeyutils1 && \
    mkdir /dpkg && \
    for deb in *.deb; do dpkg --extract $deb /dpkg || exit 10; done

FROM gcr.io/distroless/base-debian13:latest AS run

COPY --from=deb_extractor /dpkg /
