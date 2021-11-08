FROM debian:buster-slim
RUN apt-get update && \
	apt-get install -y ca-certificates && \
	update-ca-certificates && \
	rm -rf /var/lib/apt/lists/*
ADD sonic /usr/bin/sonic
RUN useradd -r -c "Sonic user" -U sonic
USER sonic
VOLUME [ "/etc/sonic" ]
WORKDIR /etc/sonic
ENTRYPOINT [ "/usr/bin/sonic" ]
CMD [ "run", "-c", "/etc/sonic/sonic.json" ]
EXPOSE 8000 8090