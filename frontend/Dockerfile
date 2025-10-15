FROM node:24-slim

WORKDIR /app

COPY package*.json ./

RUN yarn install

COPY . .

CMD [ "yarn", "run", "dev" ]