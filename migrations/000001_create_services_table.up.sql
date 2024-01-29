CREATE TABLE services (
  id INTEGER NOT NULL PRIMARY KEY,
  name TEXT NOT NULL,
  -- TODO: should be an array of strings, currently comma separated
  hosts TEXT NOT NULL,
  -- TODO: should be enumerated in a separate table
  status TEXT NOT NULL,
  container_id TEXT,
  -- TODO: should be a foreign key to images table
  image_id INTEGER NOT NULL,
  network TEXT NOT NULL,
  port TEXT,
  created DATETIME NOT NULL
);