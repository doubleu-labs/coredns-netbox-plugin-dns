FROM docker.io/netboxcommunity/netbox:v4.0

COPY ./requirements-plugin.txt /opt/netbox/
RUN /opt/netbox/venv/bin/pip install \
    --no-warn-script-location \
    -r /opt/netbox/requirements-plugin.txt

COPY configuration/plugins.py /etc/netbox/config/plugins.py
RUN SECRET_KEY="dummydummydummydummydummydummydummydummydummydummy" \
    /opt/netbox/venv/bin/python \
    /opt/netbox/netbox/manage.py \
    collectstatic --no-input

