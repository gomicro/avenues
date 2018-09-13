FROM scratch
MAINTAINER Daniel Hess <dan9186@gmail.com>

ADD avenues avenues
ADD ext/probe probe

HEALTHCHECK --interval=5s --timeout=30s --retries=3 CMD ["/probe", "http://localhost:4567/avenues/status"]

EXPOSE 4567

CMD ["/avenues"]
