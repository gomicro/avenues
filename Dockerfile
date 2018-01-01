FROM scratch
MAINTAINER Daniel Hess <dan9186@gmail.com>

ADD avenues avenues

EXPOSE 4567

CMD ["/avenues"]
