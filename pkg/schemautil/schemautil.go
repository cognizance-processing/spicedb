package schemautil

import (
	"context"

	"spicedb/pkg/datastore"

	"spicedb/internal/services/shared"
	core "spicedb/pkg/proto/core/v1"
	"spicedb/pkg/schemadsl/compiler"
)

// ValidateSchemaChanges validates the schema found in the compiled schema and returns a
// ValidatedSchemaChanges, if fully validated.
func ValidateSchemaChanges(ctx context.Context, compiled *compiler.CompiledSchema, isAdditiveOnly bool) (*shared.ValidatedSchemaChanges, error) {
	return shared.ValidateSchemaChanges(ctx, compiled, isAdditiveOnly)
}

// ApplySchemaChanges applies schema changes found in the validated changes struct, via the specified
// ReadWriteTransaction. Returns the applied changes, the validation error (if any),
// and the error itself (if any).
func ApplySchemaChanges(
	ctx context.Context,
	rwt datastore.ReadWriteTransaction,
	validated *shared.ValidatedSchemaChanges,
	existingCaveats []*core.CaveatDefinition,
	existingObjectDefs []*core.NamespaceDefinition,
) (*shared.AppliedSchemaChanges, *shared.ErrSchemaWriteDataValidation, error) {
	result, err := shared.ApplySchemaChangesOverExisting(ctx, rwt, validated, existingCaveats, existingObjectDefs)
	if err != nil {
		return result, shared.AsValidationError(err), err
	}
	return result, nil, nil
}
