package gorm

import (
	"errors"
	"fmt"
	"reflect"
)

type SQL struct {
	SQL     string
	SQLVars []interface{}
}

// First find first record that match given conditions, order by primary key
func (s *DB) FirstSQL(sql *SQL, out interface{}, where ...interface{}) *DB {
	newScope := s.clone().NewScope(out)
	newScope.Search.Limit(1)
	return newScope.Set("gorm:order_by_primary_key", "ASC").
		inlineCondition(where...).buildQuerySQL(sql).db
}

// Last find last record that match given conditions, order by primary key
func (s *DB) LastSQL(sql *SQL, out interface{}, where ...interface{}) *DB {
	newScope := s.clone().NewScope(out)
	newScope.Search.Limit(1)
	return newScope.Set("gorm:order_by_primary_key", "DESC").
		inlineCondition(where...).buildQuerySQL(sql).db
}

// Find find records that match given conditions
func (s *DB) FindSQL(sql *SQL, out interface{}, where ...interface{}) *DB {
	return s.clone().NewScope(out).inlineCondition(where...).buildQuerySQL(sql).db
}

func (s *DB) CountSQL(sql *SQL, value interface{}) *DB {
	scope := s.NewScope(s.Value)
	if query, ok := scope.Search.selects["query"]; !ok || !countingQueryRegexp.MatchString(fmt.Sprint(query)) {
		scope.Search.Select("count(*)")
	}
	scope.Search.ignoreOrderQuery = true
	result := &RowQueryResult{}
	scope.InstanceSet("row_query_result", result)

	return scope.buildRowSQL(sql).db
}

func (scope *Scope) buildQuerySQL(sql *SQL) *Scope {
	var (
		resultType reflect.Type
		results    = scope.IndirectValue()
	)

	if orderBy, ok := scope.Get("gorm:order_by_primary_key"); ok {
		if primaryField := scope.PrimaryField(); primaryField != nil {
			scope.Search.Order(fmt.Sprintf("%v.%v %v", scope.QuotedTableName(), scope.Quote(primaryField.DBName), orderBy))
		}
	}

	if value, ok := scope.Get("gorm:query_destination"); ok {
		results = indirect(reflect.ValueOf(value))
	}

	if kind := results.Kind(); kind == reflect.Slice {
		resultType = results.Type().Elem()
		results.Set(reflect.MakeSlice(results.Type(), 0, 0))

		if resultType.Kind() == reflect.Ptr {
			resultType = resultType.Elem()
		}
	} else if kind != reflect.Struct {
		scope.Err(errors.New("unsupported destination, should be slice or struct"))
		return scope
	}

	scope.prepareQuerySQL()

	if !scope.HasError() {
		scope.db.RowsAffected = 0
		if str, ok := scope.Get("gorm:query_option"); ok {
			scope.SQL += addExtraSpaceIfExist(fmt.Sprint(str))
		}

		sql.SQL = scope.SQL
		sql.SQLVars = scope.SQLVars
	}
	return scope
}

func (scope *Scope) buildRowSQL(sql *SQL) *Scope {
	if _, ok := scope.InstanceGet("row_query_result"); ok {
		scope.prepareQuerySQL()
		sql.SQL = scope.SQL
		sql.SQLVars = scope.SQLVars
	}
	return scope
}
