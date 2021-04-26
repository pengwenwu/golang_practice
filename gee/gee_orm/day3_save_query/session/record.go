package session

import (
	"geeorm/clause"
	"reflect"
)

// 多次调用clause.Set() 构造好每一个子句
// 调用一次clause.Build() 按照传入的顺序构造出最终的SQL语句
// 构造完成后，调用Raw().Exec() 方法执行
func (s *Session) Insert(values ...interface{}) (int64, error) {
	recordValues := make([]interface{}, 0)
	for _, value := range values {
		table := s.Model(value).RefTable()
		s.clause.Set(clause.INSERT, table.Name, table.FieldNames)
		recordValues = append(recordValues, table.RecordValues(value))
	}

	s.clause.Set(clause.VALUES, recordValues...)
	sql, vars := s.clause.Build(clause.INSERT, clause.VALUES)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// destSlice.Type.Elem() 获取切片的单个元素的类型destType，使用reflect.New()方法创建一个destType的实例，作为Model()的入参，映射出表结构RefTable()
// 根据表结构，使用clause构造出SELECT语句，查询到所有符合条件的记录rows
// 遍历每一行记录，利用反射创建destType的实例dest，将dest的所有字段平铺开，构造切片values
// 调用rows.Scan()将该行记录每一列的值依次赋值给values中的每一个字段
// 将dest添加到切片destSlice中。循环直到所有的记录都添加到切片destSlice中
func (s Session) Find(values interface{}) error {
	destSlice := reflect.Indirect(reflect.ValueOf(values))
	destType := destSlice.Type().Elem()
	table := s.Model(reflect.New(destType).Elem().Interface()).RefTable()

	s.clause.Set(clause.SELECT, table.Name, table.FieldNames)
	sql, vars := s.clause.Build(clause.SELECT, clause.WHERE, clause.ORDERBY, clause.LIMIT)
	rows, err := s.Raw(sql, vars...).QueryRows()
	if err != nil {
		return err
	}

	for rows.Next() {
		dest := reflect.New(destType).Elem()
		var values []interface{}
		for _, name := range table.FieldNames {
			values = append(values, dest.FieldByName(name).Addr().Interface())
		}
		if err := rows.Scan(values...); err != nil {
			return err
		}
		destSlice.Set(reflect.Append(destSlice, dest))
	}
	return rows.Close()
}