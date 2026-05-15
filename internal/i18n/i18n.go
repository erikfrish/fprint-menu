package i18n

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed locales/*.sql
var localeFS embed.FS

type Catalog struct {
	lang string
	db   *sql.DB
}

func New(requested string) (*Catalog, error) {
	lang := chooseLang(requested)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, name := range []string{"locales/en.sql", "locales/ru.sql", "locales/zh.sql"} {
		seed, err := localeFS.ReadFile(name)
		if err != nil {
			_ = db.Close()
			return nil, err
		}
		if _, err := db.Exec(string(seed)); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("load %s: %w", name, err)
		}
	}
	return &Catalog{lang: lang, db: db}, nil
}

func (c *Catalog) T(key string) string {
	var value string
	if err := c.db.QueryRow(`select value from translations where lang = ? and key = ?`, c.lang, key).Scan(&value); err == nil {
		return value
	}
	if err := c.db.QueryRow(`select value from translations where lang = 'en' and key = ?`, key).Scan(&value); err == nil {
		return value
	}
	return key
}

func (c *Catalog) Lang() string {
	return c.lang
}

func (c *Catalog) Finger(name string) string {
	if v := c.T("finger." + name); v != "finger."+name {
		return v
	}
	return name
}

func chooseLang(requested string) string {
	if lang := normalizeLang(requested); lang != "" {
		return lang
	}
	if lang := normalizeLang(os.Getenv("LANG")); lang != "" {
		return lang
	}
	return "en"
}

func normalizeLang(lang string) string {
	lang = strings.ToLower(lang)
	switch {
	case strings.HasPrefix(lang, "ru"):
		return "ru"
	case strings.HasPrefix(lang, "zh"):
		return "zh"
	case strings.HasPrefix(lang, "en"):
		return "en"
	default:
		return ""
	}
}

const schema = `
create table translations (
  lang text not null,
  key text not null,
  value text not null,
  primary key (lang, key)
);
`
