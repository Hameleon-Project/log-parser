CREATE TABLE IF NOT EXISTS logs (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50),
    nodes_count INT,
    ports_count INT,
    loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS nodes (
    id SERIAL PRIMARY KEY,
    log_id INT REFERENCES logs(id),
    name VARCHAR(255),
    type VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS ports (
    id SERIAL PRIMARY KEY,
    node_id INT REFERENCES nodes(id),
    port_number INT,
    connected_to INT REFERENCES ports(id)
);

CREATE TABLE IF NOT EXISTS links (
    id SERIAL PRIMARY KEY,
    from_port_id INT REFERENCES ports(id) ON DELETE CASCADE,
    to_port_id INT REFERENCES ports(id) ON DELETE CASCADE
);