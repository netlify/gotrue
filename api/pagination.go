package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/netlify/gotrue/models"
)

const defaultPerPage = 50

func calculateTotalPages(perPage, total uint64) uint64 {
	pages := total / perPage
	if total%perPage > 0 {
		return pages + 1
	}
	return pages
}

func addPaginationHeaders(w http.ResponseWriter, r *http.Request, p *models.Pagination) {
	totalPages := calculateTotalPages(p.PerPage, p.Count)
	url, _ := url.ParseRequestURI(r.URL.String())
	query := url.Query()
	header := ""
	if totalPages > p.Page {
		query.Set("page", fmt.Sprintf("%v", p.Page+1))
		url.RawQuery = query.Encode()
		header += "<" + url.String() + ">; rel=\"next\", "
	}
	if p.Page > 1 {
		query.Set("page", fmt.Sprintf("%v", p.Page-1))
		url.RawQuery = query.Encode()
		header += "<" + url.String() + ">; rel=\"prev\", "
	}
	query.Set("page", fmt.Sprintf("%v", totalPages))
	url.RawQuery = query.Encode()
	header += "<" + url.String() + ">; rel=\"last\""

	w.Header().Add("Link", header)
	w.Header().Add("X-Total-Count", fmt.Sprintf("%v", p.Count))
}

func paginate(r *http.Request) (*models.Pagination, error) {
	params := r.URL.Query()
	queryPage := params.Get("page")
	queryPerPage := params.Get("per_page")
	var page uint64 = 1
	var perPage uint64 = defaultPerPage
	var err error
	if queryPage != "" {
		page, err = strconv.ParseUint(queryPage, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	if queryPerPage != "" {
		perPage, err = strconv.ParseUint(queryPerPage, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return &models.Pagination{
		Page:    page,
		PerPage: perPage,
	}, nil
}
