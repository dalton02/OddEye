// Package archive busca áudio no Internet Archive (archive.org). Cada item de lá tem um .torrent
// oficial com webseed HTTP, então o download funciona mesmo sem outros peers.
package archive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const PageSize = 30

type Item struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Creator   string `json:"creator"`
	Date      string `json:"date"`
	Downloads int    `json:"downloads"`
	License   string `json:"license"`
}

type Result struct {
	Total int    `json:"total"`
	Items []Item `json:"items"`
}

type Client struct {
	BaseURL string // trocado nos testes
	HTTP    *http.Client
}

func New() *Client {
	return &Client{BaseURL: "https://archive.org", HTTP: &http.Client{Timeout: 20 * time.Second}}
}

// TorrentURL é o .torrent que o archive.org gera para todo item.
func (c *Client) TorrentURL(id string) string {
	id = url.PathEscape(id)
	return fmt.Sprintf("%s/download/%s/%s_archive.torrent", c.BaseURL, id, id)
}

// Search procura itens de áudio, os mais baixados primeiro. page começa em 1.
func (c *Client) Search(ctx context.Context, query string, page int) (Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Result{Items: []Item{}}, nil
	}
	if page < 1 {
		page = 1
	}
	q := url.Values{}
	q.Set("q", fmt.Sprintf("mediatype:audio AND (%s)", query))
	for _, f := range []string{"identifier", "title", "creator", "date", "downloads", "licenseurl"} {
		q.Add("fl[]", f)
	}
	q.Set("sort[]", "downloads desc")
	q.Set("rows", strconv.Itoa(PageSize))
	q.Set("page", strconv.Itoa(page))
	q.Set("output", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/advancedsearch.php?"+q.Encode(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "oddeye")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("não consegui falar com o archive.org: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("archive.org respondeu HTTP %d", resp.StatusCode)
	}
	var body struct {
		Response struct {
			NumFound int `json:"numFound"`
			Docs     []struct {
				Identifier string     `json:"identifier"`
				Title      flexString `json:"title"`
				Creator    flexString `json:"creator"`
				Date       flexString `json:"date"`
				Downloads  int        `json:"downloads"`
				LicenseURL flexString `json:"licenseurl"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Result{}, errors.New("resposta inesperada do archive.org")
	}
	res := Result{Total: body.Response.NumFound, Items: make([]Item, 0, len(body.Response.Docs))}
	for _, d := range body.Response.Docs {
		if d.Identifier == "" {
			continue
		}
		title := string(d.Title)
		if title == "" {
			title = d.Identifier
		}
		date := string(d.Date)
		if len(date) > 10 {
			date = date[:10]
		}
		res.Items = append(res.Items, Item{
			ID: d.Identifier, Title: title, Creator: string(d.Creator), Date: date,
			Downloads: d.Downloads, License: string(d.LicenseURL),
		})
	}
	return res, nil
}

// flexString aceita string ou lista de strings: o archive.org devolve os dois para o mesmo campo.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*f = flexString(s)
		return nil
	}
	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return nil // campo em formato desconhecido: ignora em vez de perder o item
	}
	*f = flexString(strings.Join(list, ", "))
	return nil
}
