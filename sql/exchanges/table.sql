CREATE TABLE IF NOT EXISTS exchanges (
    "id" 		  SERIAL PRIMARY KEY,
	"name" 		  TEXT NOT NULL,
	"description" TEXT NOT NULL,
	"created_on"  TIMESTAMP NOT NULL,
	"key" 	      TEXT NOT NULL
)
