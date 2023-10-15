CREATE TABLE speed_controls (
    id INTEGER PRIMARY KEY,
    district TEXT,
    location TEXT,
    created_datetime TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
