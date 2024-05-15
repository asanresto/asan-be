package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/iancoleman/strcase"
)

var conditionMap = map[string]string{
	"eq":       "=",
	"ne":       "!=",
	"in":       "IN",
	"notIn":    "NOT IN",
	"exists":   "EXISTS",
	"like":     "LIKE",
	"contains": "ILIKE",
	"gt":       ">",
	"gte":      ">=",
	"lt":       "<",
	"lte":      "<=",
}

var conditionMap2 = map[string]string{
	"Eq":       "=",
	"Ne":       "!=",
	"In":       "IN",
	"NotIn":    "NOT IN",
	"Exists":   "EXISTS",
	"Like":     "LIKE",
	"Contains": "ILIKE",
	"Gt":       ">",
	"Gte":      ">=",
	"Lt":       "<",
	"Lte":      "<=",
}

func GetPreloads(ctx context.Context, path []string) []string {
	return getNestedPreloads(
		graphql.GetOperationContext(ctx),
		graphql.CollectFieldsCtx(ctx, nil),
		"",
		path,
		0,
	)
}

// Converts an item in intersection filter array to a SQL query string.
func TranslateToQuery(obj interface{}) string {
	// Marshal to json to have the column names
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	var queryMap map[string]map[string]interface{}
	err = json.Unmarshal(jsonBytes, &queryMap)
	if err != nil {
		return ""
	}
	for key, operatorAndValue := range queryMap {
		for conditionKey, value := range operatorAndValue {
			// format := ""
			// if utils.IsNumeric(value) {
			// 	format = `"%s" %s %v`
			// } else if _, ok := value.(string); ok {
			// 	format = `"%s" %s '%s'`
			// } else {
			// 	return ""
			// }
			// return fmt.Sprintf(format, strcase.ToSnake(key), conditionMap[conditionKey], value)
			return fmt.Sprintf(`"%s" %s '%v'`, strcase.ToSnake(key), conditionMap[conditionKey], value)
		}
	}
	return ""
}

func TranslateToQuery2(obj interface{}, column *string) string {
	v := reflect.ValueOf(obj)
	// Dereference pointers if rv is a pointer to a struct
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	// Iterate over all fields of the struct
	for i := 0; i < v.NumField(); i++ {
		// Get the field type and value
		field := v.Field(i)
		if field.IsNil() {
			continue
		}
		// If the field is a pointer, dereference it to get the underlying struct value
		if field.Kind() == reflect.Pointer {
			field = field.Elem()
		}
		fieldType := v.Type().Field(i)
		// If the field is a nested struct, recursively traverse it
		if field.Kind() == reflect.Struct {
			columnKey := strcase.ToSnake(fieldType.Name)
			return TranslateToQuery2(field.Interface(), &columnKey)
		} else {
			return fmt.Sprintf(`"%s" %s '%v'`, *column, conditionMap2[fieldType.Name], field.Interface())
		}
	}
	return ""
}

func getNestedPreloads(ctx *graphql.OperationContext, fields []graphql.CollectedField, prefix string, path []string, depth int) (preloads []string) {
	for _, column := range fields {
		if column.Name == "__typename" {
			continue
		}
		var pathAtDepth = ""
		if depth < len(path) {
			pathAtDepth = path[depth]
		}
		if !strings.HasPrefix(prefix, pathAtDepth) {
			continue
		}
		prefixColumn := getPreloadString(prefix, column.Name)
		if depth >= len(path)-1 {
			preloads = append(preloads, prefixColumn)
		}
		preloads = append(preloads, getNestedPreloads(ctx, graphql.CollectFields(ctx, column.Selections, nil), prefixColumn, path, depth+1)...)
	}
	return
}

func getPreloadString(prefix, name string) string {
	if len(prefix) > 0 {
		return prefix + "." + strcase.ToSnake(name)
	}
	return strcase.ToSnake(name)
}
