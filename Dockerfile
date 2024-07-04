FROM node:18-alpine

WORKDIR /app

COPY ./package.json /app
COPY ./package-lock.json /app
COPY ./src /app/src

RUN npm install

COPY . ./

CMD ["npm", "start"]
