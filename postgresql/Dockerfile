# This Dockerfile contains the image specification of our database
# https://dev.to/danvixent/how-to-setup-postgresql-with-ssl-inside-a-docker-container-5f3
FROM postgres:15

COPY ./certs/postgresdb.key /var/lib/postgresql
COPY ./certs/postgresdb.crt /var/lib/postgresql

COPY ./certs/CA.crt /var/lib/postgresql
COPY ./certs/CA.crl /var/lib/postgresql

COPY ./ssl-conf.sh /usr/local/bin

# chown 999:999 and chmod 999 means the postgresdb.key is own by postgres
# if the owner of cert file is root, we must set chown to 0:70 and chmod to 640
RUN chown 999:999 /var/lib/postgresql/postgresdb.key && chmod 600 /var/lib/postgresql/postgresdb.key
RUN chown 999:999 /var/lib/postgresql/postgresdb.crt && chmod 600 /var/lib/postgresql/postgresdb.crt

RUN chown 999:999 /var/lib/postgresql/CA.crt && chmod 600 /var/lib/postgresql/CA.crt
RUN chown 999:999 /var/lib/postgresql/CA.crl && chmod 600 /var/lib/postgresql/CA.crl

ENTRYPOINT ["docker-entrypoint.sh"] 

CMD [ "-c", "ssl=on" , "-c", "ssl_cert_file=/var/lib/postgresql/postgresdb.crt", "-c",\
    "ssl_key_file=/var/lib/postgresql/postgresdb.key", "-c",\
    "ssl_ca_file=/var/lib/postgresql/CA.crt", "-c", "ssl_crl_file=/var/lib/postgresql/CA.crl" ]