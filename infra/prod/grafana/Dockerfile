FROM grafana/grafana:latest

# Copy files as root first
COPY grafana.ini /etc/grafana/grafana.ini
COPY provisioning/ /etc/grafana/provisioning/

# Change ownership to grafana user (UID 472 is standard for Grafana)
# The base image will switch to grafana user automatically
USER root
RUN chown -R 472:472 /etc/grafana
