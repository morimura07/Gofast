# Development
FROM node:22-slim AS dev
WORKDIR /app

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

COPY ./service-client/package.json ./service-client/package.json
COPY ./service-client/package-lock.json ./service-client/package-lock.json
RUN cd service-client && npm ci

COPY ./gen ./gen
COPY ./service-client ./service-client

CMD cd service-client && npm run dev -- --host

# Build (uses adapter-node for local/E2E testing, production deploys via wrangler CLI)
FROM node:22-slim AS prod
WORKDIR /app

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

COPY ./service-client/package.json ./service-client/package.json
COPY ./service-client/package-lock.json ./service-client/package-lock.json
RUN cd service-client && npm ci

COPY ./gen ./gen
COPY ./service-client ./service-client

ENV USE_NODE_ADAPTER=true
RUN cd service-client && npm run build

WORKDIR /app/service-client
CMD ["node", "build"]
