package finder

import "testing"

func TestInferSubject(t *testing.T) {
	cases := []struct {
		name string
		f    Filter
		want Subject
	}{
		{"has-method dominates", Filter{HasMethod: "Delete", Returns: "error"}, SubjectType},
		{"implements dominates", Filter{Implements: "Repository"}, SubjectType},
		{"has-receiver alone", Filter{HasReceiver: "OrderService"}, SubjectFunc},
		{"kind=interface alone", Filter{Kind: "interface"}, SubjectType},
		{"kind=struct alone", Filter{Kind: "struct"}, SubjectType},
		{"kind=alias alone", Filter{Kind: "alias"}, SubjectType},
		{"kind=named alone", Filter{Kind: "named"}, SubjectType},
		{"kind=func alone", Filter{Kind: "func"}, SubjectFunc},
		{"kind=method alone", Filter{Kind: "method"}, SubjectFunc},
		{"returns alone", Filter{Returns: "error"}, SubjectFunc},
		{"takes alone", Filter{Takes: "context.Context"}, SubjectFunc},
		{"takes + returns", Filter{Takes: "context.Context", Returns: "error"}, SubjectFunc},
		{"has-method + returns", Filter{HasMethod: "Delete", Returns: "error"}, SubjectType},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := InferSubject(c.f); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
