FROM scratch
MAINTAINER dev@gomicro.io

ADD avenues avenues
COPY --from=gomicro/probe /probe /probe

EXPOSE 4567

CMD ["/avenues"]
