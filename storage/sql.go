package storage

/*
func (c *Connection) UpdateOnly(model interface{}, includeColumns ...string) error {
	return c.Model(&model).Select(includeColumns).Updates(model).Error
}
*/
/*
func (c *Connection) UpdateOnly(model interface{}, values map[string]interface{}) error {
	keys := make([]string, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	return c.Model(&model).Select(keys).Updates(values).Error
}
*/
