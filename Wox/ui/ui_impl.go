package ui

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"time"
	"wox/plugin"
	"wox/util"
)

type uiImpl struct {
}

func (u *uiImpl) ChangeQuery(ctx context.Context, query string) {
	u.send(ctx, "ChangeQuery", map[string]string{
		"Query": query,
	})
}

func (u *uiImpl) HideApp(ctx context.Context) {
	u.send(ctx, "HideApp", nil)
}

func (u *uiImpl) ShowApp(ctx context.Context) {
	u.send(ctx, "ShowApp", nil)
}

func (u *uiImpl) ToggleApp(ctx context.Context) {
	u.send(ctx, "ToggleApp", nil)
}

func (u *uiImpl) ShowMsg(ctx context.Context, title string, description string, icon string) {
	u.send(ctx, "ShowMsg", map[string]string{
		"Title":       title,
		"Description": description,
		"Icon":        icon,
	})
}

func (u *uiImpl) send(ctx context.Context, method string, params map[string]string) {
	if params == nil {
		params = make(map[string]string)
	}

	util.GetLogger().Info(ctx, fmt.Sprintf("[UI] %s", method))
	requestUI(ctx, websocketRequest{
		Id:     uuid.NewString(),
		Method: method,
		Params: params,
	})
}

func onUIRequest(ctx context.Context, request websocketRequest) {
	switch request.Method {
	case "Query":
		handleQuery(ctx, request)
	case "Action":
		handleAction(ctx, request)
	case "RegisterMainHotkey":
		handleRegisterMainHotkey(ctx, request)
	}
}

func handleQuery(ctx context.Context, request websocketRequest) {
	query, ok := request.Params["query"]
	if !ok {
		logger.Error(ctx, "query parameter not found")
		responseUIError(ctx, request, "query parameter not found")
		return
	}
	if query == "" {
		responseUISuccessWithData(ctx, request, []string{})
		return
	}

	var totalResultCount int
	var startTimestamp = util.GetSystemTimestamp()
	resultChan, doneChan := plugin.GetPluginManager().Query(ctx, plugin.NewQuery(query))
	for {
		select {
		case results := <-resultChan:
			if len(results) == 0 {
				continue
			}
			totalResultCount += len(results)
			responseUISuccessWithData(ctx, request, results)
		case <-doneChan:
			logger.Info(ctx, fmt.Sprintf("query done, total results: %d, cost %d ms", totalResultCount, util.GetSystemTimestamp()-startTimestamp))
			responseUISuccessWithData(ctx, request, []string{})
			return
		case <-time.After(time.Second * 10):
			logger.Info(ctx, fmt.Sprintf("query timeout, query: %s, request id: %s", query, request.Id))
			responseUIError(ctx, request, fmt.Sprintf("query timeout, query: %s, request id: %s", query, request.Id))
			return
		}
	}

}

func handleAction(ctx context.Context, request websocketRequest) {
	resultId, ok := request.Params["id"]
	if !ok {
		logger.Error(ctx, "id parameter not found")
		responseUIError(ctx, request, "id parameter not found")
		return
	}

	action := plugin.GetPluginManager().GetAction(resultId)
	if action == nil {
		logger.Error(ctx, fmt.Sprintf("action not found for result id: %s", resultId))
		responseUIError(ctx, request, fmt.Sprintf("action not found for result id: %s", resultId))
		return
	}

	action()
	responseUISuccess(ctx, request)
}

func handleRegisterMainHotkey(ctx context.Context, request websocketRequest) {
	hotkey, ok := request.Params["hotkey"]
	if !ok {
		logger.Error(ctx, "hotkey parameter not found")
		responseUIError(ctx, request, "hotkey parameter not found")
		return
	}

	registerErr := GetUIManager().RegisterMainHotkey(ctx, hotkey)
	if registerErr != nil {
		responseUIError(ctx, request, registerErr.Error())
	} else {
		responseUISuccess(ctx, request)
	}
}
