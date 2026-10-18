FROM node:22-bookworm-slim AS verify
WORKDIR /app
COPY package.json *.mjs index.html styles.css ./
COPY tests/*.test.mjs ./tests/
RUN npm run check && npm test

FROM node:22-bookworm-slim
ENV NODE_ENV=production HOST=0.0.0.0 PORT=80 API_TARGET=http://backend:10006
WORKDIR /app
COPY --from=verify --chown=node:node /app/*.mjs /app/index.html /app/styles.css ./
USER node
EXPOSE 80
CMD ["node", "server.mjs"]
