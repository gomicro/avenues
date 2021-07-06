FROM scratch
MAINTAINER Daniel Hess <dan9186@gmail.com>

ADD avenues avenues
COPY --from=gomicro/probe /probe /probe

EXPOSE 4567

CMD ["/avenues"]
