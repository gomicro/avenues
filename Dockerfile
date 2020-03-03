FROM scratch
MAINTAINER Daniel Hess <dan9186@gmail.com>

ADD avenues avenues
ADD ext/probe probe

EXPOSE 4567

CMD ["/avenues"]
