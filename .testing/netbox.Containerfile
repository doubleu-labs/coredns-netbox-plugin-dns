FROM docker.io/netboxcommunity/netbox:v4.3.0

COPY ./requirements-plugin.txt /opt/netbox/
RUN /usr/local/bin/uv pip install \
    -r /opt/netbox/requirements-plugin.txt

COPY configuration/plugins.py /etc/netbox/config/plugins.py
RUN SECRET_KEY="dummydummydummydummydummydummydummydummydummydummy" \
    /opt/netbox/venv/bin/python \
    /opt/netbox/netbox/manage.py \
    collectstatic --no-input
