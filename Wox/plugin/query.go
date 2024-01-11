package plugin

import (
	"context"
	"github.com/samber/lo"
	"strings"
	"wox/util"
)

type QueryType = string
type QueryVariable = string

const (
	QueryTypeInput     QueryType = "input"     // user input query
	QueryTypeSelection QueryType = "selection" // user selection query
)

const (
	QueryVariableSelectedText QueryVariable = "{wox:selected_text}"
)

// Query from Wox. See "Doc/Query.md" for details.
type Query struct {
	// By default, Wox will only pass QueryTypeInput query to plugin.
	// plugin author need to enable MetadataFeatureQuerySelection feature to handle QueryTypeSelection query
	Type QueryType

	// Raw query, this includes trigger keyword if it has.
	// We didn't recommend use this property directly. You should always use Search property.
	RawQuery string

	// Trigger keyword of a query. It can be empty if user is using global trigger keyword.
	// Empty trigger keyword means this query will be a global query, see IsGlobalQuery.
	//
	// NOTE: Only available when query type is QueryTypeInput
	TriggerKeyword string

	// Command part of a query.
	// Empty command means this query doesn't have a command.
	//
	// NOTE: Only available when query type is QueryTypeInput
	Command string

	// Search part of a query.
	// Empty search means this query doesn't have a search part.
	Search string

	// User selected or drag-drop data, can be text or file or image etc
	//
	// NOTE: Only available when query type is QueryTypeSelection
	Selection util.Selection
}

func (q *Query) IsGlobalQuery() bool {
	return q.Type == QueryTypeInput && q.TriggerKeyword == ""
}

func (q *Query) String() string {
	if q.Type == QueryTypeInput {
		return q.RawQuery
	}
	if q.Type == QueryTypeSelection {
		return q.Selection.String()
	}
	return ""
}

// Query result return from plugin
type QueryResult struct {
	// Result id, should be unique. It's optional, if you don't set it, Wox will assign a random id for you
	Id string
	// Title support i18n
	Title string
	// SubTitle support i18n
	SubTitle string
	Icon     WoxImage
	Preview  WoxPreview
	Score    int64
	// Additional data associate with this result, can be retrieved in Action function
	ContextData string
	Actions     []QueryResultAction
	// refresh result after specified interval, in milliseconds. If this value is 0, Wox will not refresh this result
	// interval can only divisible by 100, if not, Wox will use the nearest number which is divisible by 100
	// E.g. if you set 123, Wox will use 200, if you set 1234, Wox will use 1300
	RefreshInterval int
	// refresh result by calling OnRefresh function
	OnRefresh func(ctx context.Context, current RefreshableResult) RefreshableResult
}

type QueryResultAction struct {
	// Result id, should be unique. It's optional, if you don't set it, Wox will assign a random id for you
	Id string
	// Name support i18n
	Name string
	Icon WoxImage
	// If true, Wox will use this action as default action. There can be only one default action in results
	// This can be omitted, if you don't set it, Wox will use the first action as default action
	IsDefault bool
	// If true, Wox will not hide after user select this result
	PreventHideAfterAction bool
	Action                 func(ctx context.Context, actionContext ActionContext)
}

type ActionContext struct {
	// Additional data associate with this result
	ContextData string
}

func (q *QueryResult) ToUI() QueryResultUI {
	return QueryResultUI{
		Id:          q.Id,
		Title:       q.Title,
		SubTitle:    q.SubTitle,
		Icon:        q.Icon,
		Preview:     q.Preview,
		Score:       q.Score,
		ContextData: q.ContextData,
		Actions: lo.Map(q.Actions, func(action QueryResultAction, index int) QueryResultActionUI {
			return QueryResultActionUI{
				Id:                     action.Id,
				Name:                   action.Name,
				Icon:                   action.Icon,
				IsDefault:              action.IsDefault,
				PreventHideAfterAction: action.PreventHideAfterAction,
			}
		}),
		RefreshInterval: q.RefreshInterval,
	}
}

type QueryResultUI struct {
	QueryId         string
	Id              string
	Title           string
	SubTitle        string
	Icon            WoxImage
	Preview         WoxPreview
	Score           int64
	ContextData     string
	Actions         []QueryResultActionUI
	RefreshInterval int
}

type QueryResultActionUI struct {
	Id                     string
	Name                   string
	Icon                   WoxImage
	IsDefault              bool
	PreventHideAfterAction bool
}

// store latest result value after query/refresh, so we can retrieve data later in action/refresh
type QueryResultCache struct {
	ResultId       string
	ResultTitle    string
	ResultSubTitle string
	ContextData    string
	Refresh        func(context.Context, RefreshableResult) RefreshableResult
	PluginInstance *Instance
	Query          Query
	Preview        WoxPreview
	Actions        *util.HashMap[string, func(ctx context.Context, actionContext ActionContext)]
}

func newQueryInputWithPlugins(query string, pluginInstances []*Instance) Query {
	var terms = strings.Split(query, " ")
	if len(terms) == 0 {
		return Query{
			Type:     QueryTypeInput,
			RawQuery: query,
		}
	}

	var rawQuery = query
	var triggerKeyword, command, search string
	var possibleTriggerKeyword = terms[0]
	var mustContainSpace = strings.Contains(query, " ")

	pluginInstance, found := lo.Find(pluginInstances, func(instance *Instance) bool {
		return lo.Contains(instance.GetTriggerKeywords(), possibleTriggerKeyword)
	})
	if found && mustContainSpace {
		// non global trigger keyword
		triggerKeyword = possibleTriggerKeyword

		if len(terms) == 1 {
			// no command and search
			command = ""
			search = ""
		} else {
			if len(terms) == 2 {
				// e.g "wpm install", we treat "install" as search, only "wpm install " will be treated as command
				command = ""
				search = terms[1]
			} else {
				var possibleCommand = terms[1]
				if lo.ContainsBy(pluginInstance.GetQueryCommands(), func(item MetadataCommand) bool {
					return item.Command == possibleCommand
				}) {
					// command and search
					command = possibleCommand
					search = strings.Join(terms[2:], " ")
				} else {
					// no command, only search
					command = ""
					search = strings.Join(terms[1:], " ")
				}
			}
		}
	} else {
		// non trigger keyword
		triggerKeyword = ""
		command = ""
		search = rawQuery
	}

	return Query{
		Type:           QueryTypeInput,
		RawQuery:       query,
		TriggerKeyword: triggerKeyword,
		Command:        command,
		Search:         search,
	}
}
