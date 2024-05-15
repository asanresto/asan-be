package directives

import (
	"asan/graph"
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
)

var CustomDirectives = graph.DirectiveRoot{
	Length: func(ctx context.Context, obj interface{}, next graphql.Resolver, min *int, max *int) (interface{}, error) {
		// value := obj.(map[string]interface{})[*graphql.GetPathContext(ctx).Field].(string)
		// if min != nil && len(value) < *min {
		// 	return nil, fmt.Errorf("length cannot be less than %d", *min)
		// }
		// if max != nil && len(value) > *max {
		// 	return nil, fmt.Errorf("length cannot be greater than %d", *max)
		// }
		return next(ctx)
	},
	Filter: func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
		// TODO: Fix this filter { where: { or: [{ and: [{ id: {} }] }] } }
		if len(obj.(map[string]interface{})) != 1 {
			return nil, errors.New("must have exactly one column, one operator and one value")
		}
		return next(ctx)
	},
}
