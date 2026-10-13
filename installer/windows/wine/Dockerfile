# A 64-bit Wine to run the Windows installer and moly.exe on Linux (test only).
FROM debian:trixie-slim
ENV DEBIAN_FRONTEND=noninteractive WINEDEBUG=-all WINEPREFIX=/home/tester/.wine WINEARCH=win64
RUN dpkg --add-architecture i386 \
 && apt-get update -qq \
 && apt-get install -y -qq --no-install-recommends wine wine64 wine32:i386 libwine libwine:i386 xvfb xauth ca-certificates procps \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -m -s /bin/bash tester
USER tester
RUN xvfb-run -a sh -c "wineboot --init; wineserver -w" >/dev/null 2>&1 || true
