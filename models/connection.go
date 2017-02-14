package models

// Namespace puts all tables names under a common
// namespace. This is useful if you want to use
// the same database for several services and don't
// want table names to collide.
var Namespace string

// Managed provides a list of models
// managed by the storage to
// perform migration tasks.
func Managed() []interface{} {
	return []interface{}{User{}, RefreshToken{}}
}

// tableName returns the name of a model's table
// in the database. It uses the namespace to isolate
// the model when this is set in the configuration.
func tableName(modelName string) string {
	if Namespace != "" {
		return Namespace + "_" + modelName
	}
	return modelName
}
