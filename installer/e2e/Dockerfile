# A real Chromium with the Moly extension, to test the whole chain: extension, native messaging, backend (Linux only).
FROM debian:trixie-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update -qq \
 && apt-get install -y -qq --no-install-recommends chromium nodejs npm xvfb xauth ca-certificates procps iproute2 python3 curl fonts-liberation \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -m -s /bin/bash tester
RUN mkdir -p /home/tester/e2e && chown tester:tester /home/tester/e2e
USER tester
WORKDIR /home/tester/e2e
RUN npm init -y >/dev/null && npm install --silent puppeteer-core@23 >/dev/null 2>&1
