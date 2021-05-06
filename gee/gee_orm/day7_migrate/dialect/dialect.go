package dialect

import "reflect"

var dialectMap = map[string]Dialect{}

// DataTypeOf 用于将Go语言的类型转换为该数据库的数据类型
// TableExistSQL 返回某个表是否存在的SQL语言，参数是表名（table）
type Dialect interface {
	DataTypeOf(typ reflect.Value) string
	TableExistSQL(tableName string) (string, []interface{})
}

func RegisterDialect(name string, dialect Dialect) {
	dialectMap[name] = dialect
}

func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectMap[name]
	return
}
