CREATE ROLE borrowhub_identity
    LOGIN
    PASSWORD 'borrowhub_identity';

CREATE DATABASE borrowhub_identity
    OWNER borrowhub_identity;

CREATE ROLE borrowhub_inventory
    LOGIN
    PASSWORD 'borrowhub_inventory';

CREATE DATABASE borrowhub_inventory
    OWNER borrowhub_inventory;

CREATE ROLE borrowhub_lending
    LOGIN
    PASSWORD 'borrowhub_lending';

CREATE DATABASE borrowhub_lending
    OWNER borrowhub_lending;

CREATE ROLE borrowhub_notification
    LOGIN
    PASSWORD 'borrowhub_notification';

CREATE DATABASE borrowhub_notification
    OWNER borrowhub_notification;