FROM golang:1.14-buster as builder

#ENV HTTP_PROXY http://proxy:port
#ENV HTTPS_PROXY http://proxy:port
ENV PACKAGE git.ops.yunlizhi.cn/devops/secrets-manager
ARG VERSION="v0.0.0"
ARG BUILD_DATE="1970-01-01T00:00:00Z"
ARG GIT_COMMIT=""
ARG GIT_TREE_STATE=""

WORKDIR /go/src/${PACKAGE}
COPY . .
RUN go build -ldflags "-X ${PACKAGE}/version.version=${VERSION} \
                        -X ${PACKAGE}/version.buildDate=${BUILD_DATE} \
                        -X ${PACKAGE}/version.gitCommit=${GIT_COMMIT} \
                        -X ${PACKAGE}/version.gitTreeState=${GIT_TREE_STATE}" -o app cmd/secrets-manager/main.go


FROM debian:buster-slim
ENV PACKAGE git.ops.yunlizhi.cn/devops/secrets-manager
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" >> /etc/timezone
ENV PORT 8000
EXPOSE 8000
WORKDIR /
COPY --from=builder /go/src/${PACKAGE}/app .
ENTRYPOINT ["/app"]
CMD ["watch", "-n", "kube-secretmanager"]
