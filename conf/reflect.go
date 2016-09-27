package conf

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const tagPrefix = "viper"

func populateConfig(config *Configuration) error {
	err := recursivelySet(reflect.ValueOf(config), "")
	if err != nil {
		return err
	}

	if config == nil {
		return nil
	}

	if config.DB.ConnURL == "" && os.Getenv("DATABASE_URL") != "" {
		config.DB.ConnURL = os.Getenv("DATABASE_URL")
	}

	if config.DB.Driver == "" && config.DB.ConnURL != "" {
		u, err := url.Parse(config.DB.ConnURL)
		if err != nil {
			return nil, errors.Wrap(err, "parsing db connection url")
		}
		config.DB.Driver = u.Scheme
	}

	if config.API.Port == 0 && os.Getenv("PORT") != "" {
		port, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return nil, errors.Wrap(err, "formatting PORT into int")
		}

		config.API.Port = port
	}

	return nil
}

func recursivelySet(val reflect.Value, prefix string) error {
	if val.Kind() != reflect.Ptr {
		return errors.Wrap(fmt.Errorf("unexpected value: %v", val), "expected pointer value")
	}

	// dereference
	val = reflect.Indirect(val)
	if val.Kind() != reflect.Struct {
		return errors.Wrap(fmt.Errorf("unexpected value: %v", val), "expected struct value")
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
