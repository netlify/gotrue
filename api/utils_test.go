package api

import "io/ioutil"

func createTestDB() string {
	f, err := ioutil.TempFile("", "test-db")
	if err != nil {
		panic(err)
	}
	return f.Name()
}
