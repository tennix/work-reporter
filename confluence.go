package main

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
	"github.com/juju/errors"
)

type Ancestor struct {
	Id string `json:"id,omitempty"`
}

type Content struct {
	Id     string `json:"id,omitempty"`
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
	Body   struct {
		Storage struct {
			Value          string `json:"value,omitempty"`
			Representation string `json:"representation,omitempty"`
		} `json:"storage,omitempty"`
	} `json:"body,omitempty"`
	Version struct {
		Number int `json:"number,omitempty"`
	} `json:"version,omitempty"`
	Links struct {
		Self  string `json:"self,omitempty"`
		WebUI string `json:"webui,omitempty"`
	} `json:"_links,omitempty"`
	Space struct {
		Key string `json:"key,omitempty"`
	} `json:"space,omitempty"`
	Ancestors []Ancestor `json:"ancestors,omitempty"`
}

type User struct {
	Type     string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
	UserKey  string `json:"userKey,omitempty"`
}

func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

func getUser(username string) User {
	apiEndpoint := "rest/api/user"
	url := fmt.Sprintf("%s?username=%s", apiEndpoint, username)
	req, err := conflunceClient.NewRequest("GET", url, nil)
	perror(errors.Trace(err))
	var user User
	_, err = conflunceClient.Do(req, &user)
	perror(errors.Trace(err))

	return user
}

func getContentByTitle(space string, title string) Content {
	opts := struct {
		Title    string `url:"title"`
		SpaceKey string `url:"spaceKey"`
		Expand   string `url:"expand"`
	}{
		Title:    title,
		SpaceKey: space,
		Expand:   "body.storage,version.number,space.key",
	}

	apiEndpoint := "rest/api/content"
	url, err := addOptions(apiEndpoint, opts)
	perror(errors.Trace(err))

	req, err := conflunceClient.NewRequest("GET", url, nil)
	perror(errors.Trace(err))

	res := struct {
		Results []Content `json:"results"`
	}{}

	_, err = conflunceClient.Do(req, &res)
	perror(errors.Trace(err))

	if len(res.Results) == 0 {
		return Content{}
	}

	return res.Results[0]
}

func getContent(id string) Content {
	apiEndpoint := fmt.Sprintf("rest/api/content/%s?expand=body.storage,version.number,space.key", id)

	req, err := conflunceClient.NewRequest("GET", apiEndpoint, nil)
	perror(errors.Trace(err))

	var content Content
	_, err = conflunceClient.Do(req, &content)
	perror(errors.Trace(err))

	return content
}

func createContent(space string, parentID string, title string, value string) Content {
	content := Content{
		Type:  "page",
		Title: title,
	}

	content.Space.Key = space
	content.Ancestors = []Ancestor{
		{Id: parentID},
	}
	content.Body.Storage.Value = value
	content.Body.Storage.Representation = "storage"

	apiEndpoint := "rest/api/content"

	req, err := conflunceClient.NewRequest("POST", apiEndpoint, &content)
	perror(errors.Trace(err))

	var respContent Content
	_, err = conflunceClient.Do(req, &respContent)
	perror(errors.Trace(err))
	return respContent
}

func updateContent(content Content, value string) Content {
	newContent := Content{
		Id:    content.Id,
		Type:  "page",
		Title: content.Title,
	}

	newContent.Space.Key = content.Space.Key
	newContent.Body.Storage.Value = value
	newContent.Body.Storage.Representation = "storage"
	newContent.Version.Number = content.Version.Number + 1

	apiEndpoint := "rest/api/content/" + content.Id

	req, err := conflunceClient.NewRequest("PUT", apiEndpoint, &newContent)
	perror(errors.Trace(err))

	var respContent Content
	_, err = conflunceClient.Do(req, &respContent)
	perror(errors.Trace(err))

	return respContent
}

func deleteContent(id string) {
	apiEndpoint := "rest/api/content/" + id

	req, err := conflunceClient.NewRequest("DELETE", apiEndpoint, nil)
	perror(errors.Trace(err))

	_, err = conflunceClient.Do(req, nil)
	perror(errors.Trace(err))
}
