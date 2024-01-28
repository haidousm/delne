CREATE TABLE services (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  -- TODO: should be an array of strings, currently comma separated
  hosts TEXT NOT NULL,
  -- TODO: should be enumerated in a separate table
  status TEXT NOT NULL,
  container_id TEXT NOT NULL,
  -- TODO: should be a foreign key to images table
  image_id INTEGER NOT NULL,
  network TEXT NOT NULL,
  port TEXT NOT NULL,
  created DATETIME NOT NULL
);