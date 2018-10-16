package namespace

var namespace string

func SetNamespace(ns string) {
	namespace = ns
}

func GetNamespace() string {
	return namespace
}
