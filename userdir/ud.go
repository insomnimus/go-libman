package userdir

import "os"
var(
userSet = os.Getenv("LIBMAN_DB_PATH")
userConfig= os.Getenv("LIBMAN_CONFIG_PATH")
)