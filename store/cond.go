package store

const (
	ColumnDeleted = "_deleted"
)

// 重新定义Cond类型，为引用类型
type Cond map[string]interface{}

func NewCond() Cond {
	return Cond{}
}

// 链式调用
func (c Cond) Where(name string, val interface{}) Cond {
	c[name] = val
	return c
}

func (c Cond) WhereNot(name string, val interface{}) Cond {
	c[name+" !="] = val
	return c
}

func (c Cond) ID(id uint64) Cond {
	c["id"] = id
	return c
}

func (c Cond) NotDeleted() Cond {
	c[ColumnDeleted] = 0
	return c
}

func (c Cond) Deleted() Cond {
	c[ColumnDeleted+" !="] = 0
	return c
}

func (c Cond) GreatThan(name string, value interface{}) Cond {
	c[name+" >"] = value
	return c
}

func (c Cond) LessThan(name string, value interface{}) Cond {
	c[name+" <"] = value
	return c
}

func (c Cond) GreatEqual(name string, value interface{}) Cond {
	c[name+" >="] = value
	return c
}

func (c Cond) LessEqual(name string, value interface{}) Cond {
	c[name+" <="] = value
	return c
}

func (c Cond) Limit(offset, limit uint) Cond {
	c["_limit"] = []uint{offset, limit}
	return c
}

func (c Cond) OrderBy(name string) Cond {
	c["_orderby"] = name
	return c
}

func (c Cond) GroupBy(name string) Cond {
	c["_groupby"] = name
	return c
}

func (c Cond) Like(name string, value interface{}) Cond {
	c[name+" like"] = value
	return c
}

func (c Cond) In(name string, values []interface{}) Cond {
	c[name+" in"] = values
	return c
}

func (c Cond) NotIn(name string, values []interface{}) Cond {
	c[name+" not in"] = values
	return c
}
