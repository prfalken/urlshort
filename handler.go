package urlshort

import (
	"log"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// MapHandler will return an http.HandlerFunc (which also
// implements http.Handler) that will attempt to map any
// paths (keys in the map) to their corresponding URL (values
// that each key in the map points to, in string format).
// If the path is not provided in the map, then the fallback
// http.Handler will be called instead.
func MapHandler(pathsToUrls map[string]string, fallback http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := pathsToUrls[r.URL.Path]; ok {
			http.Redirect(w, r, p, 302)
		} else {
			fallback.ServeHTTP(w, r)
		}
	})
}

// YAMLHandler will parse the provided YAML and then return
// an http.HandlerFunc (which also implements http.Handler)
// that will attempt to map any paths to their corresponding
// URL. If the path is not provided in the YAML, then the
// fallback http.Handler will be called instead.
//
// YAML is expected to be in the format:
//
//     - path: /some-path
//       url: https://www.some-url.com/demo
//
// The only errors that can be returned all related to having
// invalid YAML data.
//
// See MapHandler to create a similar http.HandlerFunc via
// a mapping of paths to urls.

// YAMLHandler returns a handler with redirect maps from yaml
func YAMLHandler(yaml []byte, fallback http.Handler) (http.HandlerFunc, error) {
	parsedYaml, err := parseYAML(yaml)
	if err != nil {
		return nil, err
	}
	pathMap := buildMap(parsedYaml)
	return MapHandler(pathMap, fallback), nil
}

// BoltHandler returns a handler with redirect map from BoltDB
func BoltHandler(dbFile string, fallback http.Handler) (http.HandlerFunc, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		db, err := bolt.Open(dbFile, 0600, nil)
		if err != nil {
			log.Println(err)
		}
		defer db.Close()
		var v []byte
		err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("redirects"))
			v = b.Get([]byte(r.URL.Path))
			if v == nil {
				return errors.Errorf("key not found, %v", r.URL.Path)
			}
			return nil
		})
		if err != nil {
			log.Println(err)
		}
		if string(v) == "" {
			fallback.ServeHTTP(w, r)
		} else {
			http.Redirect(w, r, string(v), 302)
		}
	}), nil
}

func parseYAML(yml []byte) (result []map[string]string, err error) {
	err = yaml.Unmarshal(yml, &result)
	if err != nil {
		return nil, err
	}
	return result, err
}

func buildMap(conf []map[string]string) map[string]string {
	resultMap := make(map[string]string)
	for _, element := range conf {
		resultMap[element["path"]] = element["url"]
	}
	return resultMap
}
