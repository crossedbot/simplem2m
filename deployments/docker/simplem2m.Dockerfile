ARG OS_NICKNAME=bullseye
ARG OS=debian:bullseye-slim
ARG ARCH=x64

FROM ${OS}

ENV SIMPLEM2M_HOME /usr/local/simplem2m
ENV PATH ${SIMPLEM2M_HOME}/bin:$PATH
RUN mkdir -vp ${SIMPLEM2M_HOME}
WORKDIR ${SIMPLEM2M_HOME}

COPY --from=simplem2m-builder /go/bin/simplem2m ./bin/simplem2m
COPY ./scripts/run.bash ./bin/run-simplem2m
COPY ./secrets/* /root/.simplem2m/

EXPOSE 8080
ENTRYPOINT [ "run-simplem2m", "-d", "mongodb://mongo:27017" ]
