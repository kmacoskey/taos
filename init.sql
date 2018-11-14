CREATE TABLE clusters (
    id               text,
    name             text,
    status           text,
    message          text,
    outputs          bytea,
    terraform_config bytea,
    terraform_state  bytea,
    timestamp        timestamp,
    expiration       timestamp,
    timeout          text,
    project          text,
    region           text
);
