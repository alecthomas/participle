package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
	"github.com/alecthomas/repr"
)

func TestExe(t *testing.T) {
	ast, err := parser.ParseString("", `
region = "us-west-2"
access_key = "something"
secret_key = "something_else"
bucket = "backups"

directory config {
    source_dir = "/etc/eventstore"
    dest_prefix = "escluster/config"
    exclude = ["*.hcl"]
    pre_backup_script = "before_backup.sh"
    post_backup_script = "after_backup.sh"
    pre_restore_script = "before_restore.sh"
    post_restore_script = "after_restore.sh"
    chmod = 0755
}

directory data {
    source_dir = "/var/lib/eventstore"
    dest_prefix = "escluster/a/data"
    exclude = [
        "*.merging"
    ]
    pre_restore_script = "before_restore.sh"
    post_restore_script = "after_restore.sh"
}
`)
	repr.Println(ast)
	require.NoError(t, err)
}
