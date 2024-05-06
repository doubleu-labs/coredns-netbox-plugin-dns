
PLUGINS = [
    'netbox_dns',
]

PLUGINS_CONFIG = {
    'netbox_dns': {
        'feature_ipam_coupling': True,
        'tolerate_underscores_in_hostnames': True,
    },
}
