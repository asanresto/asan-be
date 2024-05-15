package resolver

import (
	"asan/graph/db"
	"asan/graph/model"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
)

func CreateProduct(input model.NewProduct) (*model.Product, error) {
	// _, err := storage.Upload(context.Background(), avatarObjectName, avatar.File, avatar.Size, minio.PutObjectOptions{ContentType: avatar.ContentType})

	db.GetPostgresPool().Exec(context.Background(), `INSERT INTO "products" ("name", "description", "price") VALUES ($1, $2, $3)`, input.Name, input.Description, input.Price)
	return nil, nil
}

func Products(ctx context.Context, filter *model.ProductFiltersInput, search *string, page *int, limit *int, sort *string) (*model.PaginatedProducts, error) {
	pgxrows, count, page1, limit1, err := QueryWithCount(ctx, "products", filter, search, page, limit, sort)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(pgxrows, func(row pgx.CollectableRow) (*model.Product, error) {
		return pgx.RowToAddrOfStructByNameLax[model.Product](row)
	})
	if err != nil {
		return nil, err
	}
	return &model.PaginatedProducts{Count: &count, Items: items, Page: &page1, Limit: &limit1}, nil
}

func GenerateWhereClause(filter *model.ProductFiltersInput) string {
	if filter == nil || filter.Where == nil || len(filter.Where.Or) == 0 {
		return ""
	}
	var whereBuilder strings.Builder
	whereBuilder.WriteString("WHERE ")
	for i, or := range filter.Where.Or {
		if i == 0 {
			whereBuilder.WriteString("")
		} else {
			whereBuilder.WriteString(" OR ")
		}
		for j, and := range or.And {
			// t1 := TranslateToQuery2(*and, nil)
			// log.Println(t1)
			if query := TranslateToQuery(*and); query != "" {
				if j > 0 {
					whereBuilder.WriteString(" AND ")
				}
				whereBuilder.WriteString(query)
			} else {
				continue
			}
		}
	}
	return whereBuilder.String()
}

func QueryWithCount(
	ctx context.Context,
	table string,
	filter *model.ProductFiltersInput,
	search *string,
	page *int,
	limit *int,
	sort *string,
) (pgx.Rows, int, int, int, error) {
	var wg sync.WaitGroup
	countCh := make(chan *int)
	rowsCh := make(chan pgx.Rows)
	errCh := make(chan error)
	whereClause := GenerateWhereClause(filter)
	pg := db.GetPostgresPool()
	var limitParam int
	if limit != nil {
		limitParam = *limit
	} else {
		limitParam = 10
	}
	var pageParam int
	if page != nil {
		pageParam = *page
	} else {
		pageParam = 1
	}
	// Goroutine to get the paginated rows
	wg.Add(1)
	go func() {
		defer wg.Done()
		preloads := GetPreloads(ctx, nil)
		selected := []string{}
		for _, preload := range preloads {
			if strings.HasPrefix(preload, "items.") {
				selected = append(selected, strings.TrimPrefix(preload, "items."))
			}
		}
		var sortParam string
		if sort != nil {
			sortDirection := "ASC"
			if strings.HasPrefix(*sort, "-") {
				sortDirection = "DESC"
			}
			sortParam = fmt.Sprintf("%s %s", strings.TrimPrefix(*sort, "-"), sortDirection)
		} else {
			sortParam = "created_at DESC"
		}
		log.Println(fmt.Sprintf(`SELECT %s FROM "%s" %s ORDER BY %s LIMIT $1 OFFSET $2;`,
			strings.Join(selected[:], ","),
			table,
			whereClause,
			sortParam,
		))
		rows, err := db.GetPostgresPool().Query(context.Background(),
			fmt.Sprintf(`SELECT %s FROM "%s" %s ORDER BY %s LIMIT $1 OFFSET $2;`,
				strings.Join(selected[:], ","),
				table,
				whereClause,
				sortParam,
			),
			limitParam,
			(pageParam-1)*limitParam,
		)
		if err != nil {
			errCh <- err
			return
		}
		rowsCh <- rows
	}()
	// Goroutine to get the count of rows
	wg.Add(1)
	go func() {
		defer wg.Done()
		var count *int
		err := pg.QueryRow(context.Background(),
			fmt.Sprintf(`SELECT COUNT(*) FROM "%s" %s;`, table, whereClause),
		).Scan(&count)
		if err != nil {
			errCh <- err
			return
		}
		countCh <- count
	}()
	// Close channels when all goroutines are done
	go func() {
		wg.Wait()
		close(countCh)
		close(rowsCh)
		close(errCh)
	}()
	var count *int
	var rows pgx.Rows
	var err error
	for {
		select {
		case rows = <-rowsCh:
		case count = <-countCh:
		case err = <-errCh:
			return nil, 0, 0, 0, err
		}
		if rows != nil && count != nil {
			break
		}
	}
	return rows, *count, pageParam, limitParam, nil
}
