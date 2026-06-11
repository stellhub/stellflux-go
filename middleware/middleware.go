package middleware

import "context"

type Handler func(ctx context.Context, req any) (any, error)

type Middleware func(next Handler) Handler

type RouteInfo struct {
	Transport string
	Method    string
	Path      string
	Service   string
	Operation string
}

type Selector func(RouteInfo) bool

type Rule struct {
	Match Selector
	Use   []Middleware
}

func Chain(mws ...Middleware) Middleware {
	return func(final Handler) Handler {
		if final == nil {
			final = func(context.Context, any) (any, error) {
				return nil, nil
			}
		}
		for i := len(mws) - 1; i >= 0; i-- {
			if mws[i] == nil {
				continue
			}
			final = mws[i](final)
		}
		return final
	}
}

func MatchAll(RouteInfo) bool {
	return true
}

func MatchTransport(transport string) Selector {
	return func(info RouteInfo) bool {
		return info.Transport == transport
	}
}

func MatchPath(path string) Selector {
	return func(info RouteInfo) bool {
		return info.Path == path
	}
}

func Select(info RouteInfo, rules ...Rule) []Middleware {
	selected := make([]Middleware, 0)
	for _, rule := range rules {
		if rule.Match == nil || rule.Match(info) {
			selected = append(selected, rule.Use...)
		}
	}
	return selected
}
