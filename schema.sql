-- Applied automatically on application startup (see internal/storage/postgres.go)

CREATE TABLE IF NOT EXISTS logs (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL DEFAULT 'parsed',
    file_path TEXT,
    nodes_count INT NOT NULL DEFAULT 0,
    ports_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS nodes (
    id SERIAL PRIMARY KEY,
    log_id INT NOT NULL REFERENCES logs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    guid VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_nodes_log_id ON nodes(log_id);
CREATE INDEX IF NOT EXISTS idx_nodes_guid ON nodes(guid);

CREATE TABLE IF NOT EXISTS ports (
    id SERIAL PRIMARY KEY,
    node_id INT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    port_guid VARCHAR(255) NOT NULL,
    port_num INT NOT NULL,
    state VARCHAR(100),
    lid INT,
    msmlid INT,
    link_latency INT,
    ib_port_state INT
);

CREATE INDEX IF NOT EXISTS idx_ports_node_id ON ports(node_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ports_node_port ON ports(node_id, port_num);

CREATE TABLE IF NOT EXISTS nodes_info (
    id SERIAL PRIMARY KEY,
    node_id INT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    description TEXT,
    hardware_rev VARCHAR(100),
    serial_number VARCHAR(100),
    part_number VARCHAR(100)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_info_node_id ON nodes_info(node_id);

-- Inferred physical / logical links between ports (F-3). from_port_id < to_port_id.
CREATE TABLE IF NOT EXISTS links (
    id SERIAL PRIMARY KEY,
    log_id INT NOT NULL REFERENCES logs(id) ON DELETE CASCADE,
    from_port_id INT NOT NULL REFERENCES ports(id) ON DELETE CASCADE,
    to_port_id INT NOT NULL REFERENCES ports(id) ON DELETE CASCADE,
    kind VARCHAR(128) NOT NULL DEFAULT 'inferred',
    CONSTRAINT chk_link_port_order CHECK (from_port_id < to_port_id)
);

CREATE INDEX IF NOT EXISTS idx_links_log_id ON links(log_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_links_unique_pair ON links(log_id, from_port_id, to_port_id);

-- Upgrade older dev DB volumes created before these columns existed:
ALTER TABLE ports ADD COLUMN IF NOT EXISTS lid INT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS msmlid INT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS link_latency INT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS ib_port_state INT;
