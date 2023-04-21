package crdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"

	"spicedb/internal/datastore/common"
	pgxcommon "spicedb/internal/datastore/postgres/common"
	"spicedb/pkg/datastore"
	"spicedb/pkg/datastore/options"
	core "spicedb/pkg/proto/core/v1"
)

const (
	errUnableToReadConfig     = "unable to read namespace config: %w"
	errUnableToListNamespaces = "unable to list namespaces: %w"
)

var (
	queryReadNamespace = psql.Select(colConfig, colTimestamp)

	queryTuples = psql.Select(
		colNamespace,
		colObjectID,
		colRelation,
		colUsersetNamespace,
		colUsersetObjectID,
		colUsersetRelation,
		colCaveatContextName,
		colCaveatContext,
	)

	schema = common.NewSchemaInformation(
		colNamespace,
		colObjectID,
		colRelation,
		colUsersetNamespace,
		colUsersetObjectID,
		colUsersetRelation,
		colCaveatContextName,
		common.ExpandedLogicComparison,
	)
)

type crdbReader struct {
	txSource      pgxcommon.TxFactory
	querySplitter common.TupleQuerySplitter
	keyer         overlapKeyer
	overlapKeySet keySet
	execute       executeTxRetryFunc
	fromBuilder   func(query sq.SelectBuilder, fromStr string) sq.SelectBuilder
}

func (cr *crdbReader) ReadNamespaceByName(
	ctx context.Context,
	nsName string,
) (*core.NamespaceDefinition, datastore.Revision, error) {
	var config *core.NamespaceDefinition
	var timestamp time.Time
	if err := cr.execute(ctx, func(ctx context.Context) error {
		tx, txCleanup, err := cr.txSource(ctx)
		if err != nil {
			return err
		}
		defer txCleanup(ctx)

		config, timestamp, err = cr.loadNamespace(ctx, tx, nsName)
		if err != nil {
			if errors.As(err, &datastore.ErrNamespaceNotFound{}) {
				return err
			}
			return fmt.Errorf(errUnableToReadConfig, err)
		}

		return nil
	}); err != nil {
		return nil, datastore.NoRevision, fmt.Errorf(errUnableToReadConfig, err)
	}

	cr.addOverlapKey(config.Name)

	return config, revisionFromTimestamp(timestamp), nil
}

func (cr *crdbReader) ListAllNamespaces(ctx context.Context) ([]datastore.RevisionedNamespace, error) {
	var nsDefs []datastore.RevisionedNamespace
	if err := cr.execute(ctx, func(ctx context.Context) error {
		tx, txCleanup, err := cr.txSource(ctx)
		if err != nil {
			return err
		}
		defer txCleanup(ctx)

		nsDefs, err = loadAllNamespaces(ctx, tx, cr.fromBuilder)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errUnableToListNamespaces, err)
	}

	for _, nsDef := range nsDefs {
		cr.addOverlapKey(nsDef.Definition.Name)
	}
	return nsDefs, nil
}

func (cr *crdbReader) LookupNamespacesWithNames(ctx context.Context, nsNames []string) ([]datastore.RevisionedNamespace, error) {
	if len(nsNames) == 0 {
		return nil, nil
	}

	var nsDefs []datastore.RevisionedNamespace
	if err := cr.execute(ctx, func(ctx context.Context) error {
		tx, txCleanup, err := cr.txSource(ctx)
		if err != nil {
			return err
		}
		defer txCleanup(ctx)

		nsDefs, err = cr.lookupNamespaces(ctx, tx, nsNames)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(errUnableToListNamespaces, err)
	}

	for _, nsDef := range nsDefs {
		cr.addOverlapKey(nsDef.Definition.Name)
	}
	return nsDefs, nil
}

func (cr *crdbReader) QueryRelationships(
	ctx context.Context,
	filter datastore.RelationshipsFilter,
	opts ...options.QueryOptionsOption,
) (iter datastore.RelationshipIterator, err error) {
	query := cr.fromBuilder(queryTuples, tableTuple)
	qBuilder, err := common.NewSchemaQueryFilterer(schema, query).FilterWithRelationshipsFilter(filter)
	if err != nil {
		return nil, err
	}

	if err := cr.execute(ctx, func(ctx context.Context) error {
		iter, err = cr.querySplitter.SplitAndExecuteQuery(ctx, qBuilder, opts...)
		return err
	}); err != nil {
		return nil, err
	}

	return iter, nil
}

func (cr *crdbReader) ReverseQueryRelationships(
	ctx context.Context,
	subjectsFilter datastore.SubjectsFilter,
	opts ...options.ReverseQueryOptionsOption,
) (iter datastore.RelationshipIterator, err error) {
	query := cr.fromBuilder(queryTuples, tableTuple)
	qBuilder, err := common.NewSchemaQueryFilterer(schema, query).
		FilterWithSubjectsSelectors(subjectsFilter.AsSelector())
	if err != nil {
		return nil, err
	}

	queryOpts := options.NewReverseQueryOptionsWithOptions(opts...)

	if queryOpts.ResRelation != nil {
		qBuilder = qBuilder.
			FilterToResourceType(queryOpts.ResRelation.Namespace).
			FilterToRelation(queryOpts.ResRelation.Relation)
	}

	err = cr.execute(ctx, func(ctx context.Context) error {
		iter, err = cr.querySplitter.SplitAndExecuteQuery(
			ctx,
			qBuilder,
			options.WithLimit(queryOpts.ReverseLimit),
		)
		return err
	})

	return
}

func (cr crdbReader) loadNamespace(ctx context.Context, tx pgxcommon.DBReader, nsName string) (*core.NamespaceDefinition, time.Time, error) {
	query := cr.fromBuilder(queryReadNamespace, tableNamespace).Where(sq.Eq{colNamespace: nsName})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, time.Time{}, err
	}

	var config []byte
	var timestamp time.Time
	if err := tx.QueryRow(ctx, sql, args...).Scan(&config, &timestamp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = datastore.NewNamespaceNotFoundErr(nsName)
		}
		return nil, time.Time{}, err
	}

	loaded := &core.NamespaceDefinition{}
	if err := loaded.UnmarshalVT(config); err != nil {
		return nil, time.Time{}, err
	}

	return loaded, timestamp, nil
}

func (cr crdbReader) lookupNamespaces(ctx context.Context, tx pgxcommon.DBReader, nsNames []string) ([]datastore.RevisionedNamespace, error) {
	clause := sq.Or{}
	for _, nsName := range nsNames {
		clause = append(clause, sq.Eq{colNamespace: nsName})
	}

	query := cr.fromBuilder(queryReadNamespace, tableNamespace).Where(clause)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var nsDefs []datastore.RevisionedNamespace
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var config []byte
		var timestamp time.Time
		if err := rows.Scan(&config, &timestamp); err != nil {
			return nil, err
		}

		loaded := &core.NamespaceDefinition{}
		if err := loaded.UnmarshalVT(config); err != nil {
			return nil, fmt.Errorf(errUnableToReadConfig, err)
		}

		nsDefs = append(nsDefs, datastore.RevisionedNamespace{
			Definition:          loaded,
			LastWrittenRevision: revisionFromTimestamp(timestamp),
		})
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf(errUnableToReadConfig, rows.Err())
	}

	return nsDefs, nil
}

func loadAllNamespaces(ctx context.Context, tx pgxcommon.DBReader, fromBuilder func(sq.SelectBuilder, string) sq.SelectBuilder) ([]datastore.RevisionedNamespace, error) {
	query := fromBuilder(queryReadNamespace, tableNamespace)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var nsDefs []datastore.RevisionedNamespace
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var config []byte
		var timestamp time.Time
		if err := rows.Scan(&config, &timestamp); err != nil {
			return nil, err
		}

		loaded := &core.NamespaceDefinition{}
		if err := loaded.UnmarshalVT(config); err != nil {
			return nil, fmt.Errorf(errUnableToReadConfig, err)
		}

		nsDefs = append(nsDefs, datastore.RevisionedNamespace{
			Definition:          loaded,
			LastWrittenRevision: revisionFromTimestamp(timestamp),
		})
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf(errUnableToReadConfig, rows.Err())
	}

	return nsDefs, nil
}

func (cr *crdbReader) addOverlapKey(namespace string) {
	cr.keyer.addKey(cr.overlapKeySet, namespace)
}

var _ datastore.Reader = &crdbReader{}
