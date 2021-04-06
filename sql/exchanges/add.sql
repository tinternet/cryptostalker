INSERT INTO exchanges (
	"name",
	"description",
	"created_on",
	"key"
) VALUES (
    $1,
    $2,
    CURRENT_TIMESTAMP,
    $3
)
