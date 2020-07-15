package search

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/filtering"
	"github.com/jackbister/logsuck/internal/parser"
)

type Search struct {
	StartTime, EndTime *time.Time

	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string][]string
	NotFields    map[string][]string
	Sources      map[string]struct{}
	NotSources   map[string]struct{}
}

func Parse(searchString string, startTime, endTime *time.Time) (*Search, error) {
	res, err := parser.Parse(searchString)
	if err != nil {
		return nil, fmt.Errorf("error while parsing: %w", err)
	}

	ret := Search{
		StartTime: startTime,
		EndTime:   endTime,

		Fragments:    res.Fragments,
		NotFragments: res.NotFragments,
		Fields:       res.Fields,
		NotFields:    res.NotFields,
		Sources:      res.Sources,
		NotSources:   res.NotSources,
	}

	return &ret, nil
}

func FilterEventsStream(ctx context.Context, repo events.Repository, srch *Search, cfg *config.Config) <-chan []events.EventWithExtractedFields {
	ret := make(chan []events.EventWithExtractedFields)

	go func() {
		defer close(ret)
		inputEvents := repo.FilterStream(srch.Sources, srch.NotSources, srch.Fragments, srch.StartTime, srch.EndTime)
		compiledFrags := filtering.CompileKeys(srch.Fragments)
		compiledNotFrags := filtering.CompileKeys(srch.NotFragments)
		compiledFields := filtering.CompileMap(srch.Fields)
		compiledNotFields := filtering.CompileMap(srch.NotFields)

		for evts := range inputEvents {
			retEvts := make([]events.EventWithExtractedFields, 0)
			for _, evt := range evts {
				evtFields, include := shouldIncludeEvent(evt, cfg, compiledFrags, compiledNotFrags, compiledFields, compiledNotFields)
				if include {
					retEvts = append(retEvts, events.EventWithExtractedFields{
						Id:        evt.Id,
						Raw:       evt.Raw,
						Timestamp: evt.Timestamp,
						Source:    evt.Source,
						Fields:    evtFields,
					})
				}
			}
			ret <- retEvts
		}
	}()
	return ret
}

func shouldIncludeEvent(evt events.EventWithId,
	cfg *config.Config,
	compiledFrags []*regexp.Regexp, compiledNotFrags []*regexp.Regexp,
	compiledFields map[string][]*regexp.Regexp, compiledNotFields map[string][]*regexp.Regexp) (map[string]string, bool) {
	rawLowered := strings.ToLower(evt.Raw)
	evtFields := parser.ExtractFields(strings.ToLower(evt.Raw), cfg.FieldExtractors)
	// TODO: This could produce unexpected results
	evtFields["source"] = evt.Source

	include := true
	for _, frag := range compiledFrags {
		if !frag.MatchString(rawLowered) {
			include = false
			break
		}
	}
	for _, frag := range compiledNotFrags {
		if frag.MatchString(rawLowered) {
			include = false
			break
		}
	}
	for key, values := range compiledFields {
		evtValue, ok := evtFields[key]
		if !ok {
			include = false
			break
		}
		anyMatch := false
		for _, value := range values {
			if value.MatchString(evtValue) {
				anyMatch = true
			}
		}
		if !anyMatch {
			include = false
			break
		}
	}
	for key, values := range compiledNotFields {
		evtValue, ok := evtFields[key]
		if !ok {
			break
		}
		anyMatch := false
		for _, value := range values {
			if value.MatchString(evtValue) {
				anyMatch = true
			}
		}
		if anyMatch {
			include = false
			break
		}
	}
	return evtFields, include
}
