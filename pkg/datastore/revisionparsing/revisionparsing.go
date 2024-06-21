package revisionparsing

import (
	"spicedb/internal/datastore/crdb"
	"spicedb/internal/datastore/memdb"
	"spicedb/internal/datastore/mysql"
	"spicedb/internal/datastore/postgres"
	"spicedb/internal/datastore/revisions"
	"spicedb/internal/datastore/spanner"
	"spicedb/pkg/datastore"
)

// ParsingFunc is a function that can parse a string into a revision.
type ParsingFunc func(revisionStr string) (rev datastore.Revision, err error)

// ParseRevisionStringByDatastoreEngineID defines a map from datastore engine ID to its associated
// revision parsing function.
var ParseRevisionStringByDatastoreEngineID = map[string]ParsingFunc{
	memdb.Engine:    ParsingFunc(memdb.ParseRevisionString),
	crdb.Engine:     ParsingFunc(crdb.ParseRevisionString),
	postgres.Engine: ParsingFunc(postgres.ParseRevisionString),
	mysql.Engine:    ParsingFunc(mysql.ParseRevisionString),
	spanner.Engine:  ParsingFunc(spanner.ParseRevisionString),
}

// MustParseRevisionForTest is a convenience ParsingFunc that can be used in tests and panics when parsing an error.
func MustParseRevisionForTest(revisionStr string) (rev datastore.Revision) {
	rev, err := testParser(revisionStr)
	if err != nil {
		panic(err)
	}

	return rev
}

var testParser = revisions.RevisionParser(revisions.HybridLogicalClock)
