package main

import "encoding/json"

type host struct {
	Domain       string        `json:"domain"`
	Index        bool          `json:"index"`
	Repositories *repositories `json:"repositories"`
}

type repository struct {
	Name       string     `json:"json"`
	Prefix     string     `json:"prefix"`
	Subs       []sub      `json:"subs"`
	URL        string     `json:"url"`
	Main       bool       `json:"main"`
	SourceURLs sourceURLs `json:"source"`
	Website    website    `json:"website"`
}

type repositories []repository

func (rs *repositories) append(r repository) {
	(*rs) = append(*rs, r)
}

type sourceURLs struct {
	Home string `json:"home"`
	Dir  string `json:"dir"`
	File string `json:"file"`
}

type website struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type sub struct {
	Name   string
	Hidden bool
}

func (s *sub) UnmarshalJSON(raw []byte) error {
	*s = sub{}

	err := json.Unmarshal(raw, &s.Name)
	if err == nil {
		return nil
	}

	subWithTags := struct {
		Name   string `json:"name"`
		Hidden bool   `json:"hidden"`
	}{}
	err = json.Unmarshal(raw, &subWithTags)
	if err != nil {
		return err
	}
	*s = sub(subWithTags)
	return nil
}
