FROM amd64/golang:1.13-alpine AS build

WORKDIR src/github.com/argoproj/argo-events
COPY . .
COPY assets vendor/github.com/argoproj/argo-cd
RUN apk add --update build-base git \
    && make resource-linux \
    && mv dist/resource-gateway /bin/
    
ENTRYPOINT [ "bin/resource-gateway" ]