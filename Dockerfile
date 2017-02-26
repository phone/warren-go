FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
ADD slacktoken.txt /
ADD warren-linux /
CMD ["/warren-go-linux"]
