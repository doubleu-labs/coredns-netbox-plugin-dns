PLUGINS = [
    'netbox_dns',
]

PLUGINS_CONFIG = {
    'netbox_dns': {
        'tolerate_underscores_in_labels': True,
    },
}

FIELD_CHOICES = {
    'netbox_dns.Zone.status+': [
        ('catalog', 'Catalog', 'orange'),
    ],
}
