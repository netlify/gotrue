package conf

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/spf13/viper"
)

const tagPrefix = "viper"

func populateConfig(config *Configuration) error {
	return recursivelySet(reflect.ValueOf(config), "")
}

func recursivelySet(val reflect.Value, prefix string) error {
	if val.Kind() != reflect.Ptr {
		return errors.New("WTF")
	}

	// dereference
	val = reflect.Indirect(val)
	if val.Kind() != reflect.Struct {
		return errors.New("FML")
	}

	// grab the type for this instance
	vType := reflect.TypeOf(val.Interface())

	// go through child fields
	for i := 0; i < val.NumField(); i++ {
		thisField := val.Field(i)
		thisType := vType.Field(i)
		tag := prefix + getTag(thisType)

		switch thisField.Kind() {
		case reflect.Struct:
			recursivelySet(thisField.Addr(), tag+".")
		case reflect.Int:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			// you can only set with an int64 -> int
			configVal := int64(viper.GetInt(tag))
			thisField.SetInt(configVal)
		case reflect.String:
			configVal := viper.GetString(tag)
			thisField.SetString(configVal)
		default:
			return fmt.Errorf("unexpected type detected ~ aborting: %s", thisField.Kind())
		}
	}

	return nil
}

func getTag(field reflect.StructField) string {
	// check if maybe we have a special magic tag
	tag := field.Tag
	if tag != "" {
		for _, prefix := range []string{tagPrefix, "mapstructure", "json"} {
			if v := tag.Get(prefix); v != "" {
				return v
			}
		}
	}

	return field.Name
}
