FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY web/ ./
RUN yarn build

FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o caslingo .

FROM alpine:3.22
# git clones the projects' repositories and pushes the translations
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY --from=backend /src/caslingo ./
COPY --from=backend /src/conf ./conf
COPY --from=web /web/build ./web/build
ENV dbName=/app/data/caslingo.db reposDir=/app/data/repos filesDir=/app/data/files
VOLUME /app/data
EXPOSE 16500
ENTRYPOINT ["./caslingo"]
