/**
type Image struct {
	ID         int
	Repository string
	Name       string
	Tag        string

	Created string
}
*/

CREATE TABLE images (
  id SERIAL PRIMARY KEY,
  repository TEXT NOT NULL,
  name TEXT NOT NULL,
  tag TEXT NOT NULL,
  created DATETIME NOT NULL
);