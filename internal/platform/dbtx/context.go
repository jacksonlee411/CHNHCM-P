// Package dbtx binds an already-open physical transaction at infrastructure
// boundaries. Public ports carry context and values, never persistence handles.
package dbtx

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type key struct{}

func With(ctx context.Context, tx pgx.Tx) context.Context { return context.WithValue(ctx, key{}, tx) }
func From(ctx context.Context) (pgx.Tx, bool)             { tx, ok := ctx.Value(key{}).(pgx.Tx); return tx, ok }
