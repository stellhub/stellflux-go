package boot

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	stdhttp "net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	cacheclient "github.com/stellhub/stellar/clients/cache"
	"github.com/stellhub/stellar/config"
	apperrors "github.com/stellhub/stellar/errors"
	stellarhttp "github.com/stellhub/stellar/transport/http"
)

type redisDebugAPI struct {
	app    *App
	prefix string
}

type mysqlDebugAPI struct {
	app        *App
	prefix     string
	tableMu    sync.Mutex
	tableReady bool
}

type postgresqlDebugAPI struct {
	app        *App
	prefix     string
	tableMu    sync.Mutex
	tableReady bool
}

type cacheDebugAPI struct {
	app    *App
	prefix string
}

type redisDebugItemRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   string `json:"ttl"`
}

type mysqlDebugItemRequest struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type mysqlDebugItemResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type postgresqlDebugItemRequest struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type postgresqlDebugItemResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type cacheDebugItemRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type cacheDebugItemResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func registerRedisDebugAPI(app *App, cfg *config.RedisConfig) {
	if app == nil || cfg == nil || !debugAPIEnabled(cfg.DebugAPI) {
		return
	}
	api := redisDebugAPI{
		app:    app,
		prefix: debugAPIPrefix(cfg.DebugAPI, "/redis"),
	}
	router := app.HTTP()
	router.POST(debugPath(api.prefix, "/items"), api.setItem)
	router.GET(debugPath(api.prefix, "/items"), api.getItem)
	router.PUT(debugPath(api.prefix, "/items"), api.setItem)
	router.DELETE(debugPath(api.prefix, "/items"), api.deleteItem)
	router.GET(debugPath(api.prefix, "/keys"), api.listKeys)
}

func registerMySQLDebugAPI(app *App, cfg *config.MySQLConfig) {
	if app == nil || cfg == nil || !debugAPIEnabled(cfg.DebugAPI) {
		return
	}
	api := &mysqlDebugAPI{
		app:    app,
		prefix: debugAPIPrefix(cfg.DebugAPI, "/mysql"),
	}
	router := app.HTTP()
	router.POST(debugPath(api.prefix, "/items"), api.createItem)
	router.GET(debugPath(api.prefix, "/items"), api.getItem)
	router.PUT(debugPath(api.prefix, "/items"), api.updateItem)
	router.DELETE(debugPath(api.prefix, "/items"), api.deleteItem)
	router.GET(debugPath(api.prefix, "/items/list"), api.listItems)
}

func registerPostgreSQLDebugAPI(app *App, cfg *config.PostgreSQLConfig) {
	if app == nil || cfg == nil || !debugAPIEnabled(cfg.DebugAPI) {
		return
	}
	api := &postgresqlDebugAPI{
		app:    app,
		prefix: debugAPIPrefix(cfg.DebugAPI, "/postgresql"),
	}
	router := app.HTTP()
	router.POST(debugPath(api.prefix, "/items"), api.createItem)
	router.GET(debugPath(api.prefix, "/items"), api.getItem)
	router.PUT(debugPath(api.prefix, "/items"), api.updateItem)
	router.DELETE(debugPath(api.prefix, "/items"), api.deleteItem)
	router.GET(debugPath(api.prefix, "/items/list"), api.listItems)
}

func registerCacheDebugAPI(app *App, cfg *config.CacheConfig) {
	if app == nil || cfg == nil || !debugAPIEnabled(cfg.DebugAPI) {
		return
	}
	api := cacheDebugAPI{
		app:    app,
		prefix: debugAPIPrefix(cfg.DebugAPI, "/cache"),
	}
	router := app.HTTP()
	router.POST(debugPath(api.prefix, "/items"), api.setItem)
	router.GET(debugPath(api.prefix, "/items"), api.getItem)
	router.PUT(debugPath(api.prefix, "/items"), api.setItem)
	router.DELETE(debugPath(api.prefix, "/items"), api.deleteItem)
	router.GET(debugPath(api.prefix, "/stats"), api.stats)
}

func (api redisDebugAPI) setItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	client, err := api.redisClient()
	if err != nil {
		return nil, err
	}

	var body redisDebugItemRequest
	if err := decodeDebugJSON(request, &body); err != nil {
		return nil, err
	}
	if body.Key == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "redis key is required", stdhttp.StatusBadRequest)
	}

	ttl := time.Duration(0)
	if body.TTL != "" {
		parsed, err := time.ParseDuration(body.TTL)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeInvalidArgument, "invalid redis ttl", err, stdhttp.StatusBadRequest)
		}
		ttl = parsed
	}
	if err := client.Set(ctx, body.Key, body.Value, ttl).Err(); err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"key": body.Key,
		"ttl": body.TTL,
	}), nil
}

func (api redisDebugAPI) getItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	client, err := api.redisClient()
	if err != nil {
		return nil, err
	}

	key := request.Query.Get("key")
	if key == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "redis key is required", stdhttp.StatusBadRequest)
	}
	value, err := client.Get(ctx, key).Result()
	if err != nil {
		if stderrors.Is(err, goredis.Nil) {
			return nil, apperrors.New(apperrors.CodeNotFound, "redis key not found", stdhttp.StatusNotFound)
		}
		return nil, err
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"key":   key,
		"value": value,
		"ttl":   ttl.String(),
	}), nil
}

func (api redisDebugAPI) deleteItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	client, err := api.redisClient()
	if err != nil {
		return nil, err
	}

	key := request.Query.Get("key")
	if key == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "redis key is required", stdhttp.StatusBadRequest)
	}
	deleted, err := client.Del(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"key":     key,
		"deleted": deleted,
	}), nil
}

func (api redisDebugAPI) listKeys(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	client, err := api.redisClient()
	if err != nil {
		return nil, err
	}

	pattern := request.Query.Get("pattern")
	if pattern == "" {
		pattern = "*"
	}
	limit := int64(20)
	if value := request.Query.Get("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return nil, apperrors.New(apperrors.CodeInvalidArgument, "redis key limit must be a positive integer", stdhttp.StatusBadRequest)
		}
		limit = parsed
	}
	keys, _, err := client.Scan(ctx, 0, pattern, limit).Result()
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"pattern": pattern,
		"keys":    keys,
	}), nil
}

func (api redisDebugAPI) redisClient() (*goredis.Client, error) {
	client, ok := api.app.RedisClient()
	if !ok || client == nil {
		return nil, apperrors.New(apperrors.CodeInternal, "redis client is not configured", stdhttp.StatusInternalServerError)
	}
	return client, nil
}

func (api cacheDebugAPI) setItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	store, err := api.cache()
	if err != nil {
		return nil, err
	}

	var body cacheDebugItemRequest
	if err := decodeDebugJSON(request, &body); err != nil {
		return nil, err
	}
	if strings.TrimSpace(body.Key) == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "cache key is required", stdhttp.StatusBadRequest)
	}
	if err := store.SetString(ctx, body.Key, body.Value); err != nil {
		return nil, err
	}

	status := stdhttp.StatusOK
	if request.Method == stdhttp.MethodPost {
		status = stdhttp.StatusCreated
	}
	return stellarhttp.JSON(status, cacheDebugItemResponse{
		Key:   body.Key,
		Value: body.Value,
	}), nil
}

func (api cacheDebugAPI) getItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	store, err := api.cache()
	if err != nil {
		return nil, err
	}
	key := request.Query.Get("key")
	if strings.TrimSpace(key) == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "cache key is required", stdhttp.StatusBadRequest)
	}

	value, ok, err := store.GetString(ctx, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperrors.New(apperrors.CodeNotFound, "cache key not found", stdhttp.StatusNotFound)
	}
	return stellarhttp.JSON(stdhttp.StatusOK, cacheDebugItemResponse{
		Key:   key,
		Value: value,
	}), nil
}

func (api cacheDebugAPI) deleteItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	store, err := api.cache()
	if err != nil {
		return nil, err
	}
	key := request.Query.Get("key")
	if strings.TrimSpace(key) == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "cache key is required", stdhttp.StatusBadRequest)
	}

	deleted, err := store.Delete(ctx, key)
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"key":     key,
		"deleted": deleted,
	}), nil
}

func (api cacheDebugAPI) stats(context.Context, *stellarhttp.Request) (*stellarhttp.Response, error) {
	store, err := api.cache()
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"adapter":  store.AdapterName(),
		"entries":  store.Len(),
		"capacity": store.Capacity(),
	}), nil
}

func (api cacheDebugAPI) cache() (*cacheclient.Cache, error) {
	store, ok := api.app.Cache()
	if !ok || store == nil {
		return nil, apperrors.New(apperrors.CodeInternal, "cache is not configured", stdhttp.StatusInternalServerError)
	}
	return store, nil
}

func (api *mysqlDebugAPI) createItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.mysqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	var body mysqlDebugItemRequest
	if err := decodeDebugJSON(request, &body); err != nil {
		return nil, err
	}
	if body.Name == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "mysql item name is required", stdhttp.StatusBadRequest)
	}

	result, err := db.ExecContext(ctx, "INSERT INTO stellar_example_items (name, value) VALUES (?, ?)", body.Name, body.Value)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusCreated, mysqlDebugItemResponse{
		ID:    id,
		Name:  body.Name,
		Value: body.Value,
	}), nil
}

func (api *mysqlDebugAPI) getItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.mysqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	id, err := queryDebugID(request, "id")
	if err != nil {
		return nil, err
	}
	item, err := api.selectItem(ctx, db, id)
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, item), nil
}

func (api *mysqlDebugAPI) updateItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.mysqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	var body mysqlDebugItemRequest
	if err := decodeDebugJSON(request, &body); err != nil {
		return nil, err
	}
	if body.ID <= 0 {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "mysql item id is required", stdhttp.StatusBadRequest)
	}
	if body.Name == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "mysql item name is required", stdhttp.StatusBadRequest)
	}

	result, err := db.ExecContext(ctx, "UPDATE stellar_example_items SET name = ?, value = ? WHERE id = ?", body.Name, body.Value, body.ID)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, apperrors.New(apperrors.CodeNotFound, "mysql item not found", stdhttp.StatusNotFound)
	}
	return stellarhttp.JSON(stdhttp.StatusOK, mysqlDebugItemResponse{
		ID:    body.ID,
		Name:  body.Name,
		Value: body.Value,
	}), nil
}

func (api *mysqlDebugAPI) deleteItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.mysqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	id, err := queryDebugID(request, "id")
	if err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, "DELETE FROM stellar_example_items WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"id":      id,
		"deleted": affected,
	}), nil
}

func (api *mysqlDebugAPI) listItems(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.mysqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	limit := 20
	if value := request.Query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return nil, apperrors.New(apperrors.CodeInvalidArgument, "mysql item limit must be a positive integer", stdhttp.StatusBadRequest)
		}
		limit = parsed
	}

	rows, err := db.QueryContext(ctx, "SELECT id, name, value FROM stellar_example_items ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []mysqlDebugItemResponse{}
	for rows.Next() {
		var item mysqlDebugItemResponse
		if err := rows.Scan(&item.ID, &item.Name, &item.Value); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"items": items,
	}), nil
}

func (api *mysqlDebugAPI) ensureTable(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	api.tableMu.Lock()
	defer api.tableMu.Unlock()
	if api.tableReady {
		return nil
	}
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS stellar_example_items (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(128) NOT NULL,
	value TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`)
	if err != nil {
		return err
	}
	api.tableReady = true
	return nil
}

func (api *mysqlDebugAPI) selectItem(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id int64) (mysqlDebugItemResponse, error) {
	var item mysqlDebugItemResponse
	err := db.QueryRowContext(ctx, "SELECT id, name, value FROM stellar_example_items WHERE id = ?", id).
		Scan(&item.ID, &item.Name, &item.Value)
	if err == nil {
		return item, nil
	}
	if err == sql.ErrNoRows {
		return item, apperrors.New(apperrors.CodeNotFound, "mysql item not found", stdhttp.StatusNotFound)
	}
	return item, err
}

func (api *mysqlDebugAPI) mysqlDB() (interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, error) {
	db, ok := api.app.MySQLDB()
	if !ok || db == nil {
		return nil, apperrors.New(apperrors.CodeInternal, "mysql db is not configured", stdhttp.StatusInternalServerError)
	}
	return db, nil
}

func (api *postgresqlDebugAPI) createItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.postgresqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	var body postgresqlDebugItemRequest
	if err := decodeDebugJSON(request, &body); err != nil {
		return nil, err
	}
	if body.Name == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "postgresql item name is required", stdhttp.StatusBadRequest)
	}

	var id int64
	if err := db.QueryRowContext(ctx, "INSERT INTO stellar_example_items (name, value) VALUES ($1, $2) RETURNING id", body.Name, body.Value).Scan(&id); err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusCreated, postgresqlDebugItemResponse{
		ID:    id,
		Name:  body.Name,
		Value: body.Value,
	}), nil
}

func (api *postgresqlDebugAPI) getItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.postgresqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	id, err := queryDebugID(request, "id")
	if err != nil {
		return nil, err
	}
	item, err := api.selectItem(ctx, db, id)
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, item), nil
}

func (api *postgresqlDebugAPI) updateItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.postgresqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	var body postgresqlDebugItemRequest
	if err := decodeDebugJSON(request, &body); err != nil {
		return nil, err
	}
	if body.ID <= 0 {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "postgresql item id is required", stdhttp.StatusBadRequest)
	}
	if body.Name == "" {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "postgresql item name is required", stdhttp.StatusBadRequest)
	}

	result, err := db.ExecContext(ctx, "UPDATE stellar_example_items SET name = $1, value = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3", body.Name, body.Value, body.ID)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, apperrors.New(apperrors.CodeNotFound, "postgresql item not found", stdhttp.StatusNotFound)
	}
	return stellarhttp.JSON(stdhttp.StatusOK, postgresqlDebugItemResponse{
		ID:    body.ID,
		Name:  body.Name,
		Value: body.Value,
	}), nil
}

func (api *postgresqlDebugAPI) deleteItem(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.postgresqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	id, err := queryDebugID(request, "id")
	if err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, "DELETE FROM stellar_example_items WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"id":      id,
		"deleted": affected,
	}), nil
}

func (api *postgresqlDebugAPI) listItems(ctx context.Context, request *stellarhttp.Request) (*stellarhttp.Response, error) {
	db, err := api.postgresqlDB()
	if err != nil {
		return nil, err
	}
	if err := api.ensureTable(ctx, db); err != nil {
		return nil, err
	}

	limit := 20
	if value := request.Query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return nil, apperrors.New(apperrors.CodeInvalidArgument, "postgresql item limit must be a positive integer", stdhttp.StatusBadRequest)
		}
		limit = parsed
	}

	rows, err := db.QueryContext(ctx, "SELECT id, name, value FROM stellar_example_items ORDER BY id DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []postgresqlDebugItemResponse{}
	for rows.Next() {
		var item postgresqlDebugItemResponse
		if err := rows.Scan(&item.ID, &item.Name, &item.Value); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stellarhttp.JSON(stdhttp.StatusOK, map[string]any{
		"items": items,
	}), nil
}

func (api *postgresqlDebugAPI) ensureTable(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	api.tableMu.Lock()
	defer api.tableMu.Unlock()
	if api.tableReady {
		return nil
	}
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS stellar_example_items (
	id BIGSERIAL PRIMARY KEY,
	name VARCHAR(128) NOT NULL,
	value TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		return err
	}
	api.tableReady = true
	return nil
}

func (api *postgresqlDebugAPI) selectItem(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id int64) (postgresqlDebugItemResponse, error) {
	var item postgresqlDebugItemResponse
	err := db.QueryRowContext(ctx, "SELECT id, name, value FROM stellar_example_items WHERE id = $1", id).
		Scan(&item.ID, &item.Name, &item.Value)
	if err == nil {
		return item, nil
	}
	if err == sql.ErrNoRows {
		return item, apperrors.New(apperrors.CodeNotFound, "postgresql item not found", stdhttp.StatusNotFound)
	}
	return item, err
}

func (api *postgresqlDebugAPI) postgresqlDB() (interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, error) {
	db, ok := api.app.PostgreSQLDB()
	if !ok || db == nil {
		return nil, apperrors.New(apperrors.CodeInternal, "postgresql db is not configured", stdhttp.StatusInternalServerError)
	}
	return db, nil
}

func queryDebugID(request *stellarhttp.Request, name string) (int64, error) {
	value := request.Query.Get(name)
	if value == "" {
		return 0, apperrors.New(apperrors.CodeInvalidArgument, "mysql item id is required", stdhttp.StatusBadRequest)
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperrors.New(apperrors.CodeInvalidArgument, "mysql item id must be a positive integer", stdhttp.StatusBadRequest)
	}
	return id, nil
}

func decodeDebugJSON(request *stellarhttp.Request, target any) error {
	if request.Body == nil {
		return apperrors.New(apperrors.CodeInvalidArgument, "request body is required", stdhttp.StatusBadRequest)
	}
	defer request.Body.Close()
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return apperrors.Wrap(apperrors.CodeInvalidArgument, "invalid JSON request body", err, stdhttp.StatusBadRequest)
	}
	return nil
}

func debugAPIEnabled(cfg *config.DebugAPIConfig) bool {
	if cfg == nil || cfg.Enabled == nil {
		return false
	}
	return *cfg.Enabled
}

func debugAPIPrefix(cfg *config.DebugAPIConfig, fallback string) string {
	if cfg != nil && strings.TrimSpace(cfg.Prefix) != "" {
		return cfg.Prefix
	}
	return fallback
}

func debugPath(prefix string, path string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "/"
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}
